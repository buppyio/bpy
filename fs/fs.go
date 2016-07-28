package fs

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/htree"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

type DirEnts []DirEnt

type DirEnt struct {
	EntName    string
	EntSize    int64
	EntModTime int64
	EntMode    os.FileMode
	Data       [32]byte
}

func (ent *DirEnt) Name() string       { return ent.EntName }
func (ent *DirEnt) Size() int64        { return ent.EntSize }
func (ent *DirEnt) Mode() os.FileMode  { return ent.EntMode }
func (ent *DirEnt) ModTime() time.Time { return time.Unix(ent.EntModTime, 0) }
func (ent *DirEnt) IsDir() bool        { return ent.EntMode.IsDir() }
func (ent *DirEnt) Sys() interface{}   { return nil }

func (dir DirEnts) Len() int           { return len(dir) }
func (dir DirEnts) Less(i, j int) bool { return dir[i].EntName < dir[j].EntName }
func (dir DirEnts) Swap(i, j int)      { dir[i], dir[j] = dir[j], dir[i] }

func WriteDir(store bpy.CStoreWriter, indir DirEnts, mode os.FileMode) (DirEnt, error) {
	dir := make(DirEnts, len(indir)+1, len(indir)+1)
	copy(dir[1:], indir)
	mode |= os.ModeDir
	ent := DirEnt{EntName: ".", EntMode: mode}
	dir[0] = ent
	sort.Sort(dir[1:])
	for i := 1; i < len(dir)-1; i++ {
		if dir[i].EntName == "." {
			return DirEnt{}, fmt.Errorf("cannot name file or folder '.'")
		}
		if dir[i].EntName == dir[i+1].EntName {
			return DirEnt{}, fmt.Errorf("duplicate directory entry '%s'", dir[i].EntName)
		}
	}

	nbytes := 0
	for i := range dir {
		nbytes += 2 + len(dir[i].EntName) + 8 + 4 + 8 + 32
	}

	buf := bytes.NewBuffer(make([]byte, 0, nbytes))
	for _, e := range dir {
		var buffer [8]byte
		if len(e.EntName) > 65535 {
			return DirEnt{}, fmt.Errorf("directory entry name '%s' too long", e.EntName)
		}
		binary.LittleEndian.PutUint16(buffer[0:2], uint16(len(e.EntName)))
		// err is always nil for buf writes, no need to check.
		buf.Write(buffer[0:2])
		buf.WriteString(e.EntName)

		binary.LittleEndian.PutUint64(buffer[0:8], uint64(e.EntSize))
		buf.Write(buffer[0:8])

		binary.LittleEndian.PutUint32(buffer[0:4], uint32(e.EntMode))
		buf.Write(buffer[0:4])

		binary.LittleEndian.PutUint64(buffer[0:8], uint64(e.EntModTime))
		buf.Write(buffer[0:8])

		buf.Write(e.Data[:])
	}

	tw := htree.NewWriter(store)
	_, err := tw.Write(buf.Bytes())
	if err != nil {
		return DirEnt{}, err
	}

	data, err := tw.Close()
	if err != nil {
		return DirEnt{}, err
	}
	ent.Data = data
	return ent, nil
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
			EntName:    name,
			EntSize:    size,
			EntMode:    mode,
			EntModTime: modtime,
			Data:       hash,
		})
	}
	// fill in the hash for "."
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
		if entname == "" {
			entname = "."
		}
		ents, err := ReadDir(store, hash)
		if err != nil {
			return result, err
		}
		found := false
		j := 0
		for j = range ents {
			if ents[j].EntName == entname {
				found = true
				break
			}
		}
		if !found {
			return result, fmt.Errorf("no such directory: %s", entname)
		}
		if i != end-1 {
			if !ents[j].EntMode.IsDir() {
				return result, fmt.Errorf("not a directory: %s", ents[j].EntName)
			}
			hash = ents[j].Data
		} else {
			result = ents[j]
		}
	}
	if result.EntName == "." {
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
	case io.SeekStart:
		o, err := r.rdr.Seek(uint64(off))
		r.offset = o
		return int64(o), err
	case io.SeekCurrent:
		o, err := r.rdr.Seek(r.offset + uint64(off))
		r.offset = o
		return int64(o), err
	case io.SeekEnd:
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

func (r *FileReader) ReadAt(buf []byte, off int64) (int, error) {
	if r.offset != uint64(off) {
		_, err := r.Seek(off, io.SeekStart)
		if err != nil {
			return 0, err
		}
	}
	return io.ReadFull(r, buf)
}

func (r *FileReader) Close() error {
	// nothing to do but having Close in the api isn't bad
	// if we need to add it.
	return nil
}

func Open(store bpy.CStoreReader, roothash [32]byte, fpath string) (*FileReader, error) {
	dirent, err := Walk(store, roothash, fpath)
	if err != nil {
		return nil, err
	}
	if dirent.EntMode.IsDir() {
		return nil, fmt.Errorf("%s is a directory", fpath)
	}
	rdr, err := htree.NewReader(store, dirent.Data)
	if err != nil {
		return nil, err
	}
	return &FileReader{
		offset: 0,
		fsize:  dirent.EntSize,
		rdr:    rdr,
	}, nil
}

func Ls(store bpy.CStoreReader, roothash [32]byte, fpath string) (DirEnts, error) {
	dirent, err := Walk(store, roothash, fpath)
	if err != nil {
		return nil, err
	}
	if !dirent.EntMode.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", fpath)
	}
	ents, err := ReadDir(store, dirent.Data)
	if err != nil {
		return nil, err
	}
	return ents, nil
}

func EmptyDir(store bpy.CStoreWriter, mode os.FileMode) (DirEnt, error) {
	return WriteDir(store, []DirEnt{}, mode)
}

func Insert(rstore bpy.CStoreReader, wstore bpy.CStoreWriter, dest [32]byte, destPath string, ent DirEnt) (DirEnt, error) {
	if destPath == "" || destPath[0] != '/' {
		destPath = "/" + destPath
	}
	if !strings.HasSuffix(destPath, "/") {
		ent.EntName = path.Base(destPath)
		destPath = path.Dir(destPath)
	}
	destPath = path.Clean(destPath)
	pathElems := strings.Split(destPath, "/")[1:]
	if pathElems[len(pathElems)-1] == "" {
		pathElems = pathElems[:len(pathElems)-1]
	}
	return insert(rstore, wstore, dest, pathElems, ent)
}

func insert(rstore bpy.CStoreReader, wstore bpy.CStoreWriter, dest [32]byte, destPath []string, ent DirEnt) (DirEnt, error) {
	destEnts, err := ReadDir(rstore, dest)
	if err != nil {
		return DirEnt{}, err
	}
	if len(destPath) == 0 {
		mode := destEnts[0].EntMode
		// Reuse '.' entry for new entry
		destEnts[0] = ent
		return WriteDir(wstore, destEnts, mode)
	}
	for i := 0; i < len(destEnts); i++ {
		if destEnts[i].EntName == destPath[0] {
			if !destEnts[i].IsDir() {
				return DirEnt{}, fmt.Errorf("%s is not a directory", destEnts[i].EntName)
			}
			newEnt, err := insert(rstore, wstore, destEnts[i].Data, destPath[1:], ent)
			if err != nil {
				return DirEnt{}, err
			}
			destEnts[i].Data = newEnt.Data
			return WriteDir(wstore, destEnts[1:], destEnts[0].EntMode)
		}
	}
	return DirEnt{}, fmt.Errorf("no folder or file '%s'", destPath[0])
}

func Remove(rstore bpy.CStoreReader, wstore bpy.CStoreWriter, root [32]byte, filePath string) (DirEnt, error) {
	if filePath == "" || filePath[0] != '/' {
		filePath = "/" + filePath
	}
	filePath = path.Clean(filePath)
	if filePath == "/" {
		return DirEnt{}, errors.New("cannot remove root directory")
	}
	pathElems := strings.Split(filePath, "/")[1:]
	if pathElems[len(pathElems)-1] == "" {
		pathElems = pathElems[:len(pathElems)-1]
	}
	return remove(rstore, wstore, root, pathElems)
}

func remove(rstore bpy.CStoreReader, wstore bpy.CStoreWriter, root [32]byte, filePath []string) (DirEnt, error) {
	dirEnts, err := ReadDir(rstore, root)
	if err != nil {
		return DirEnt{}, err
	}
	if len(filePath) == 1 {
		mode := dirEnts[0].EntMode
		dirEnts = dirEnts[1:]
		nents := len(dirEnts) - 1
		newDir := make(DirEnts, 0, nents)
		for i := 0; i < len(dirEnts); i++ {
			if filePath[0] != dirEnts[i].EntName {
				newDir = append(newDir, dirEnts[i])
			}
		}
		if len(newDir) > nents {
			return DirEnt{}, fmt.Errorf("no file called '%s'", filePath[0])
		}
		return WriteDir(wstore, newDir, mode)
	}
	for i := 0; i < len(dirEnts); i++ {
		if dirEnts[i].EntName == filePath[0] {
			if !dirEnts[i].IsDir() {
				return DirEnt{}, fmt.Errorf("%s is not a directory", dirEnts[i].EntName)
			}
			newEnt, err := remove(rstore, wstore, dirEnts[i].Data, filePath[1:])
			if err != nil {
				return DirEnt{}, err
			}
			dirEnts[i].Data = newEnt.Data
			return WriteDir(wstore, dirEnts[1:], dirEnts[0].EntMode)
		}
	}
	return DirEnt{}, fmt.Errorf("no folder or file '%s'", filePath[0])
}

func Copy(rstore bpy.CStoreReader, wstore bpy.CStoreWriter, root [32]byte, destPath, srcPath string) (DirEnt, error) {
	srcEnt, err := Walk(rstore, root, srcPath)
	if err != nil {
		return DirEnt{}, err
	}
	newRoot, err := Insert(rstore, wstore, root, destPath, srcEnt)
	if err != nil {
		return DirEnt{}, err
	}
	return newRoot, nil
}

func Move(rstore bpy.CStoreReader, wstore bpy.CStoreWriter, root [32]byte, destPath, srcPath string) (DirEnt, error) {
	copyRoot, err := Copy(rstore, wstore, root, destPath, srcPath)
	if err != nil {
		return DirEnt{}, err
	}
	newRoot, err := Remove(rstore, wstore, copyRoot.Data, srcPath)
	if err != nil {
		return DirEnt{}, err
	}
	return newRoot, nil
}
