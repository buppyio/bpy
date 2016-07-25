package fsutil

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/htree"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func hostFileToHashTree(store bpy.CStoreWriter, path string) ([32]byte, error) {
	fin, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer fin.Close()
	fout := htree.NewWriter(store)
	if err != nil {
		return [32]byte{}, err
	}
	_, err = io.Copy(fout, fin)
	if err != nil {
		return [32]byte{}, err
	}
	return fout.Close()
}

func CpHostDirToFs(store bpy.CStoreWriter, path string) (fs.DirEnt, error) {
	st, err := os.Stat(path)
	if err != nil {
		return fs.DirEnt{}, err
	}
	ents, err := ioutil.ReadDir(path)
	if err != nil {
		return fs.DirEnt{}, err
	}
	dir := make(fs.DirEnts, 0, 16)
	for _, e := range ents {
		switch {
		case e.Mode().IsRegular():
			hash, err := hostFileToHashTree(store, filepath.Join(path, e.Name()))
			if err != nil {
				return fs.DirEnt{}, err
			}
			dir = append(dir, fs.DirEnt{
				EntName: e.Name(),
				Data:    hash,
				EntSize: e.Size(),
				EntMode: e.Mode(),
			})
		case e.IsDir():
			newEnt, err := CpHostDirToFs(store, filepath.Join(path, e.Name()))
			if err != nil {
				return fs.DirEnt{}, err
			}
			dir = append(dir, fs.DirEnt{
				EntName: e.Name(),
				EntMode: e.Mode(),
				Data:    newEnt.Data,
			})
		}
	}
	return fs.WriteDir(store, dir, st.Mode())
}

func CpHashTreeToHostFile(store bpy.CStoreReader, hash [32]byte, dst string, mode os.FileMode) error {
	f, err := htree.NewReader(store, hash)
	if err != nil {
		return err
	}
	fout, err := os.OpenFile(dst, os.O_EXCL|os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(fout, f)
	if err != nil {
		_ = fout.Close()
		return err
	}
	return fout.Close()

}

func CpFsDirToHost(store bpy.CStoreReader, hash [32]byte, dest string) error {
	ents, err := fs.ReadDir(store, hash)
	if err != nil {
		return err
	}
	err = os.Mkdir(dest, ents[0].EntMode)
	if err != nil {
		return err
	}
	for _, e := range ents[1:] {
		subp := filepath.Join(dest, e.EntName)
		switch {
		case e.EntMode.IsDir():
			err = CpFsDirToHost(store, e.Data, subp)
			if err != nil {
				return err
			}
		case e.EntMode.IsRegular():
			err = CpHashTreeToHostFile(store, e.Data, subp, e.EntMode)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
