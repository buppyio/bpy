package fs

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/htree"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
)

type DirEnts []DirEnt

type DirEnt struct {
	Name    string
	Size    int64
	ModTime int64
	Mode    os.FileMode
	Data    [32]byte
}

func (dir DirEnts) Len() int           { return len(dir) }
func (dir DirEnts) Less(i, j int) bool { return dir[i].Name < dir[j].Name }
func (dir DirEnts) Swap(i, j int)      { dir[i], dir[j] = dir[j], dir[i] }

func WriteDir(store bpy.CStoreWriter, indir DirEnts, mode os.FileMode) ([32]byte, error) {
	var numbytes [8]byte
	var dirBuf [256]DirEnt
	var dir DirEnts

	// Best effort at stack allocating this slice
	// XXX todo benchmark the affect of this.
	// XXX should probably factor code so it doesn't need to do this copy
	if len(indir)+1 < len(dirBuf) {
		dir = dirBuf[0 : len(indir)+1]
	} else {
		dir = make(DirEnts, len(indir)+1, len(indir)+1)
	}
	copy(dir[1:], indir)
	mode |= os.ModeDir
	dir[0] = DirEnt{Name: "", Mode: mode}

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

func ReadDir(store bpy.CStoreReader, hash [32]byte) (DirEnts, error) {
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
	// Fill current directory "" by sorted order must be the first entry.
	dir[0].Data = hash
	return dir, nil
}

func Walk(store bpy.CStoreReader, hash [32]byte, fpath string) (DirEnt, error) {
	var result DirEnt
	var end int

	if fpath == "" || fpath[0] != '/' {
		fpath = "/" + fpath
	}
	fpath = path.Clean(fpath)
	pathelems := strings.Split(fpath, "/")
	if pathelems[len(pathelems)-1] == "" {
		end = len(pathelems) - 1
	} else {
		end = len(pathelems)
	}
	for i := 0; i < end; i++ {
		entname := pathelems[i]
		ents, err := ReadDir(store, hash)
		if err != nil {
			return result, err
		}
		found := false
		j := 0
		for j = range ents {
			if ents[j].Name == entname {
				found = true
				break
			}
		}
		if !found {
			return result, fmt.Errorf("no such directory: %s", entname)
		}

		if i != end-1 {
			if !ents[j].Mode.IsDir() {
				fmt.Errorf("not a directory: %s", ents[j].Name)
			}
			hash = ents[j].Data
		} else {
			result = ents[j]
		}
	}
	if result.Name == "" {
		result.Data = hash
	}
	return result, nil
}

type FileReader struct {
	offset uint64
	fsize  int64
	rdr    *htree.Reader
}

func (r *FileReader) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case 0:
		o, err := r.rdr.Seek(uint64(off))
		r.offset = o
		return int64(o), err
	case 1:
		o, err := r.rdr.Seek(r.offset + uint64(off))
		r.offset = o
		return int64(o), err
	case 2:
		o, err := r.rdr.Seek(uint64(r.fsize + off))
		r.offset = o
		return int64(o), err
	default:
		return int64(r.offset), fmt.Errorf("bad whence %d", whence)
	}
}

func (r *FileReader) Read(buf []byte) (int, error) {
	nread, err := r.rdr.Read(buf)
	r.offset += uint64(nread)
	return nread, err
}

func Open(store bpy.CStoreReader, roothash [32]byte, fpath string) (*FileReader, error) {
	dirent, err := Walk(store, roothash, fpath)
	if err != nil {
		return nil, err
	}
	if dirent.Mode.IsDir() {
		return nil, fmt.Errorf("%s is a directory", fpath)
	}
	rdr, err := htree.NewReader(store, dirent.Data)
	if err != nil {
		return nil, err
	}
	return &FileReader{
		offset: 0,
		fsize:  dirent.Size,
		rdr:    rdr,
	}, nil
}

func Ls(store bpy.CStoreReader, roothash [32]byte, fpath string) (DirEnts, error) {
	dirent, err := Walk(store, roothash, fpath)
	if err != nil {
		return nil, err
	}
	if !dirent.Mode.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", fpath)
	}
	ents, err := ReadDir(store, dirent.Data)
	if err != nil {
		return nil, err
	}
	return ents[1:], nil
}
