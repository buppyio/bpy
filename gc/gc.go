package gc

import (
	"bufio"
	"errors"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/bpack"
	"github.com/buppyio/bpy/cstore/cache"
	"github.com/buppyio/bpy/drive"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/remote/client"
	// "log"
	"path"
	"sort"
)

type gcState struct {
	gcGeneration uint64
	k            *bpy.Key
	c            *client.Client
	store        bpy.CStore
	cache        *cache.Client
	visited      map[[32]byte]struct{}

	// Sweeping state
	newPackSize uint64
	newPack     *bpack.Writer
	moved       map[[32]byte]struct{}
	canDelete   []string
}

func GC(c *client.Client, store bpy.CStore, cacheClient *cache.Client, k *bpy.Key) error {
	gcGeneration, err := remote.StartGC(c)
	if err != nil {
		return err
	}

	// Doing this twice shouldn't hurt if theres a premature error.
	defer remote.StopGC(c)

	gc := &gcState{
		cache:        cacheClient,
		gcGeneration: gcGeneration,
		k:            k,
		c:            c,
		store:        store,
		visited:      make(map[[32]byte]struct{}),
		moved:        make(map[[32]byte]struct{}),
		newPack:      nil,
		newPackSize:  0,
		canDelete:    []string{},
	}

	hash, _, ok, err := remote.GetRoot(gc.c, gc.k)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("root missing")
	}

	err = gc.markRef(hash)
	if err != nil {
		return err
	}

	err = store.Close()
	if err != nil {
		return err
	}

	err = gc.sweep()
	if err != nil {
		return err
	}

	return remote.StopGC(c)
}

func (gc *gcState) markRef(hash [32]byte) error {
	err := gc.markHTree(hash)
	if err != nil {
		return err
	}

	ref, err := refs.GetRef(gc.store, hash)
	if err != nil {
		return err
	}

	err = gc.markFsDir(ref.Root)
	if err != nil {
		return err
	}

	if ref.HasPrev {
		err = gc.markRef(ref.Prev)
		if err != nil {
			return err
		}
	}
	return nil
}

func (gc *gcState) markFsDir(root [32]byte) error {
	err := gc.markHTree(root)
	if err != nil {
		return err
	}
	dirEnts, err := fs.ReadDir(gc.store, root)
	if err != nil {
		return err
	}
	for _, dirEnt := range dirEnts[1:] {
		if dirEnt.IsDir() {
			err := gc.markFsDir(dirEnt.HTree.Data)
			if err != nil {
				return err
			}
		}
		if dirEnt.HTree.Depth == 0 {
			gc.visited[dirEnt.HTree.Data] = struct{}{}
		} else {
			err := gc.markHTree(dirEnt.HTree.Data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (gc *gcState) markHTree(root [32]byte) error {
	_, ok := gc.visited[root]
	if ok {
		return nil
	}

	data, err := gc.store.Get(root)
	if err != nil {
		return err
	}

	if data[0] == 0 {
		gc.visited[root] = struct{}{}
		return nil
	}

	for i := 1; i < len(data); i += 40 {
		var hash [32]byte
		copy(hash[:], data[i+8:i+40])
		if data[0] == 1 {
			gc.visited[hash] = struct{}{}
		}
		err := gc.markHTree(hash)
		if err != nil {
			return err
		}
	}
	gc.visited[root] = struct{}{}
	return nil
}

func (gc *gcState) putValue(hash [32]byte, val []byte) error {
	_, moved := gc.moved[hash]
	if moved {
		return nil
	}

	if gc.newPackSize+uint64(len(val))+uint64(len(hash)) > 128*1024*1024 {
		err := gc.closeCurrentWriterAndDeleteOldPacks()
		if err != nil {
			return err
		}
	}

	if gc.newPack == nil {
		name, err := bpy.RandomFileName()
		if err != nil {
			return err
		}
		f, err := gc.c.NewPack(path.Join("packs", name) + ".ebpack")
		if err != nil {
			return err
		}
		buffered := &bpy.BufferedWriteCloser{
			W: f,
			B: bufio.NewWriterSize(f, 65536),
		}
		gc.newPack, err = bpack.NewEncryptedWriter(buffered, gc.k.CipherKey)
		if err != nil {
			return err
		}
	}

	err := gc.newPack.Add(string(hash[:]), val)
	if err != nil {
		return err
	}
	// Only approximate, but good enough.
	gc.newPackSize += uint64(len(hash)) + uint64(len(val))
	gc.moved[hash] = struct{}{}
	return nil
}

func (gc *gcState) closeCurrentWriterAndDeleteOldPacks() error {
	if gc.newPack != nil {
		_, err := gc.newPack.Close()
		if err != nil {
			return err
		}
	}
	gc.newPack = nil
	gc.newPackSize = 0
	for _, toDelete := range gc.canDelete {
		err := remote.Remove(gc.c, toDelete, gc.gcGeneration)
		if err != nil {
			return err
		}
	}
	gc.canDelete = []string{}
	return nil
}

func (gc *gcState) sweep() error {
	packs, err := remote.ListPacks(gc.c)
	if err != nil {
		return err
	}
	for _, pack := range packs {
		err = gc.sweepPack(pack)
		if err != nil {
			return err
		}
	}
	err = gc.closeCurrentWriterAndDeleteOldPacks()
	if err != nil {
		return err
	}
	return nil
}

type offsetSortedIdx []bpack.IndexEnt

func (idx offsetSortedIdx) Len() int           { return len(idx) }
func (idx offsetSortedIdx) Swap(i, j int)      { idx[i], idx[j] = idx[j], idx[i] }
func (idx offsetSortedIdx) Less(i, j int) bool { return idx[i].Offset < idx[j].Offset }

func (gc *gcState) sweepPack(pack drive.PackListing) error {
	packPath := path.Join("packs/", pack.Name)
	f, err := gc.c.Open(packPath)
	if err != nil {
		return err
	}
	packReader, err := bpack.NewEncryptedReader(f, gc.k.CipherKey, int64(pack.Size))
	if err != nil {
		return err
	}
	defer packReader.Close()
	// XXX fetch from local cache if we have it.
	err = packReader.ReadIndex()
	if err != nil {
		return err
	}

	idx := offsetSortedIdx(packReader.Idx)
	sort.Sort(idx)

	if pack.Size > 100*1024*1024 {
		canSkip := true
		for _, idxEnt := range idx {
			var hash [32]byte
			copy(hash[:], idxEnt.Key)
			_, ok := gc.visited[hash]
			if !ok {
				// log.Printf("can't skip 1, %s", hex.EncodeToString(hash[:]))
				canSkip = false
				break
			}
			_, ok = gc.moved[hash]
			if ok {
				// log.Printf("can't skip 2")
				canSkip = false
				break
			}
		}

		// We can totally skip this pack if its full, reachable and has no duplicates.
		if canSkip {
			for _, idxEnt := range idx {
				var hash [32]byte
				copy(hash[:], idxEnt.Key)
				gc.moved[hash] = struct{}{}
			}
			return nil
		}
	}

	for i := 0; i < len(idx); i++ {
		var hash [32]byte
		copy(hash[:], idx[i].Key)
		if gc.cache != nil {
			val, ok, err := gc.cache.GetRaw(hash)
			if err != nil {
				return err
			}
			if ok {
				err = gc.putValue(hash, val)
				if err != nil {
					return err
				}
				continue
			}
		}

		// Find the longest consecutive 'run' of values we must fetch
		run := []bpack.IndexEnt{}
		runSize := uint32(0)
		for i < len(idx) {
			copy(hash[:], idx[i].Key)
			_, isReachable := gc.visited[hash]
			if !isReachable {
				break
			}
			_, ok := gc.moved[hash]
			if ok {
				break
			}
			run = append(run, idx[i])
			runSize += idx[i].Size
			i++
		}
		if len(run) == 0 {
			continue
		}
		runBase := run[0].Offset
		// log.Printf("moving run of values: base=%v, size=%v", runBase, runSize)
		runData, err := packReader.GetAt(runBase, runSize)
		if err != nil {
			return err
		}

		for _, idxEnt := range run {
			var hash [32]byte
			copy(hash[:], idxEnt.Key)
			val := runData[0:idxEnt.Size]
			runData = runData[idxEnt.Size:]
			err = gc.putValue(hash, val)
			if err != nil {
				return err
			}
			if gc.cache != nil {
				err = gc.cache.PutRaw(hash, val)
				if err != nil {
					return err
				}
			}
		}
	}
	gc.canDelete = append(gc.canDelete, packPath)
	return nil
}
