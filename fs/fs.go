package fs

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/htree"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
)

type DirEnts []DirEnt

type DirEnt struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime int64
	Data    [32]byte
}

func (dir DirEnts) Len() int           { return len(dir) }
func (dir DirEnts) Less(i, j int) bool { return dir[i].Name < dir[j].Name }
func (dir DirEnts) Swap(i, j int)      { dir[i], dir[j] = dir[j], dir[i] }

func WriteDir(store bpy.CStore, dir DirEnts) ([32]byte, error) {
	var numbytes [8]byte

	sort.Sort(dir)
	for i := 0; i < len(dir)-1; i++ {
		if dir[i].Name == dir[i+1].Name {
			return [32]byte{}, fmt.Errorf("duplicate directory entry '%s'", dir[i].Name)
		}
	}

	nbytes := 0
	for i := range dir {
		nbytes += 2 + len(dir[i].Name) + 8 + 4 + 8 + 32
	}
	buf := bytes.NewBuffer(make([]byte, 0, nbytes))
	for _, e := range dir {
		// err is always nil for buf writes, no need to check.
		if len(e.Name) > 65535 {
			return [32]byte{}, fmt.Errorf("directory entry name '%s' too long", e.Name)
		}
		binary.LittleEndian.PutUint16(numbytes[0:2], uint16(len(e.Name)))
		buf.Write(numbytes[0:2])
		buf.WriteString(e.Name)

		binary.LittleEndian.PutUint64(numbytes[0:8], uint64(e.Size))
		buf.Write(numbytes[0:8])

		binary.LittleEndian.PutUint32(numbytes[0:4], uint32(e.Mode))
		buf.Write(numbytes[0:4])

		binary.LittleEndian.PutUint64(numbytes[0:8], uint64(e.ModTime))
		buf.Write(numbytes[0:8])

		buf.Write(e.Data[:])
	}

	tw := htree.NewWriter(store)
	_, err := tw.Write(buf.Bytes())
	if err != nil {
		return [32]byte{}, err
	}

	return tw.Close()
}

func ReadDir(store bpy.CStore, hash [32]byte) (DirEnts, error) {
	var dir DirEnts
	rdr, err := htree.NewReader(store, hash)
	if err != nil {
		return nil, err
	}
	dirdata, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}
	for len(dirdata) != 0 {
		var hash [32]byte
		namelen := int(binary.LittleEndian.Uint16(dirdata[0:2]))
		dirdata = dirdata[2:]
		name := string(dirdata[0:namelen])
		dirdata = dirdata[namelen:]
		size := int64(binary.LittleEndian.Uint64(dirdata[0:8]))
		dirdata = dirdata[8:]
		mode := os.FileMode(binary.LittleEndian.Uint32(dirdata[0:4]))
		dirdata = dirdata[4:]
		modtime := int64(binary.LittleEndian.Uint64(dirdata[0:8]))
		dirdata = dirdata[8:]
		copy(hash[:], dirdata[0:32])
		dirdata = dirdata[32:]
		dir = append(dir, DirEnt{
			Name:    name,
			Size:    size,
			Mode:    mode,
			ModTime: modtime,
			Data:    hash,
		})
	}
	return dir, nil
}
