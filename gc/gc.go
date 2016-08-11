package gc

import (
	"bufio"
	"errors"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/bpack"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/remote/client"
	"log"
	"path"
	"sort"
)

func GC(c *client.Client, store bpy.CStore, k *bpy.Key) error {
	gcId, err := remote.StartGC(c)
	if err != nil {
		return err
	}
	// Doing this twice shouldn't hurt if theres a premature error.
	defer remote.StopGC(c)

	reachable := make(map[[32]byte]struct{})
	err = markRef(c, store, "default", reachable)
	if err != nil {
		return err
	}

	err = sweep(c, k, reachable, gcId)
	if err != nil {
		return err
	}

	return remote.StopGC(c)
}

func markRef(c *client.Client, store bpy.CStore, ref string, visited map[[32]byte]struct{}) error {
	tag, ok, err := remote.GetTag(c, ref)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("tag does not exist")
	}
	root, err := bpy.ParseHash(tag)
	if err != nil {
		return err
	}

	err = markFsDir(store, root, visited)
	if err != nil {
		return err
	}
	return nil
}

func markFsDir(store bpy.CStore, root [32]byte, visited map[[32]byte]struct{}) error {
	err := markHTree(store, root, visited)
	if err != nil {
		return err
	}
	dirEnts, err := fs.ReadDir(store, root)
	if err != nil {
		return err
	}
	for _, dirEnt := range dirEnts[1:] {
		if dirEnt.IsDir() {
			err := markFsDir(store, dirEnt.Data, visited)
			if err != nil {
				return err
			}
		}
		err := markHTree(store, dirEnt.Data, visited)
		if err != nil {
			return err
		}
	}
	return nil
}

func markHTree(store bpy.CStore, root [32]byte, visited map[[32]byte]struct{}) error {
	_, ok := visited[root]
	if ok {
		return nil
	}
	data, err := store.Get(root)
	if err != nil {
		return err
	}
	if data[0] == 0 {
		visited[root] = struct{}{}
		return nil
	}

	for i := 1; i < len(data); i += 40 {
		var hash [32]byte
		copy(hash[:], data[i+8:i+40])
		if data[0] == 1 {
			visited[hash] = struct{}{}
		} else {
			err := markHTree(store, hash, visited)
			if err != nil {
				return err
			}
		}
	}
	visited[root] = struct{}{}
	return nil
}

type offsetSortedIdx []bpack.IndexEnt

func (idx offsetSortedIdx) Len() int           { return len(idx) }
func (idx offsetSortedIdx) Swap(i, j int)      { idx[i], idx[j] = idx[j], idx[i] }
func (idx offsetSortedIdx) Less(i, j int) bool { return idx[i].Offset < idx[j].Offset }

func sweep(c *client.Client, k *bpy.Key, reachable map[[32]byte]struct{}, gcId string) error {
	packs, err := remote.ListPacks(c)
	if err != nil {
		return err
	}
	moved := make(map[[32]byte]struct{})
	var newPack *bpack.Writer
	newPackSize := uint64(0)
	canDelete := []string{}
	for _, pack := range packs {
		packPath := path.Join("packs/", pack.Name)
		f, err := c.Open(packPath)
		if err != nil {
			return err
		}
		packReader, err := bpack.NewEncryptedReader(f, k.CipherKey, int64(pack.Size))
		if err != nil {
			return err
		}
		err = packReader.ReadIndex()
		if err != nil {
			return err
		}
		idx := offsetSortedIdx(packReader.Idx)
		sort.Sort(idx)

		if pack.Size > 120*1024*1024 {
			canSkip := true
			for _, idxEnt := range idx {
				var hash [32]byte
				copy(hash[:], idxEnt.Key)
				_, ok := reachable[hash]
				if !ok {
					canSkip = false
					break
				}
				_, ok = moved[hash]
				if ok {
					canSkip = false
					break
				}
			}
			if canSkip {
				continue
			}
		}

		for i := 0; i < len(idx); i++ {
			run := []bpack.IndexEnt{}
			for i < len(idx) {
				var hash [32]byte
				copy(hash[:], idx[i].Key)
				_, isReachable := reachable[hash]
				if !isReachable {
					break
				}
				_, ok := moved[hash]
				if ok {
					break
				}
				run = append(run, idx[i])
				i++
			}
			if len(run) == 0 {
				continue
			}

			runBase := run[0].Offset
			runSize := uint32(run[len(run)-1].Offset) + run[len(run)-1].Size - uint32(run[0].Offset)
			runData, err := packReader.GetAt(runBase, runSize)
			if err != nil {
				return err
			}
			log.Printf("runBase=%v,runSize=%v", runBase, runSize)
			for _, idxEnt := range run {
				var hash [32]byte
				copy(hash[:], idxEnt.Key)
				if newPackSize+uint64(idxEnt.Size) > 128*1024*1024 {
					_, err := newPack.Close()
					if err != nil {
						return err
					}
					newPack = nil
					newPackSize = 0
					for _, toDelete := range canDelete {
						err := remote.Remove(c, toDelete, gcId)
						if err != nil {
							return err
						}
					}
					canDelete = []string{}
				}
				if newPack == nil {
					name, err := bpy.RandomFileName()
					if err != nil {
						return err
					}
					f, err := c.NewPack(path.Join("packs", name) + ".ebpack")
					if err != nil {
						return err
					}
					buffered := &bpy.BufferedWriteCloser{
						W: f,
						B: bufio.NewWriterSize(f, 65536),
					}
					newPack, err = bpack.NewEncryptedWriter(buffered, k.CipherKey)
					if err != nil {
						return err
					}
				}
				val := runData[0:idxEnt.Size]
				runData = runData[idxEnt.Size:]
				err = newPack.Add(idxEnt.Key, val)
				if err != nil {
					return err
				}
				// Only approximate, but good enough.
				newPackSize += uint64(len(idxEnt.Key)) + uint64(len(val))
				moved[hash] = struct{}{}
			}
		}
		err = packReader.Close()
		if err != nil {
			return err
		}
		canDelete = append(canDelete, packPath)
	}
	if newPack != nil {
		_, err = newPack.Close()
		if err != nil {
			return err
		}
	}
	for _, toDelete := range canDelete {
		err := remote.Remove(c, toDelete, gcId)
		if err != nil {
			return err
		}
	}
	return nil
}
