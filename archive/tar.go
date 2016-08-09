package archive

import (
	"archive/tar"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/fs"
	"io"
	"path"
)

func Tar(store bpy.CStore, dirHash [32]byte, out io.Writer) error {
	tw := tar.NewWriter(out)
	err := writeTar(store, "", dirHash, tw)
	if err != nil {
		return err
	}
	return tw.Close()
}

func writeTar(store bpy.CStore, curpath string, dirHash [32]byte, out *tar.Writer) error {
	ents, err := fs.ReadDir(store, dirHash)
	if err != nil {
		return err
	}
	for _, ent := range ents[1:] {
		if ent.IsDir() {
			err = writeTar(store, path.Join(curpath, ent.EntName), ent.Data, out)
			if err != nil {
				return err
			}
			continue
		}
		f, err := fs.Open(store, ents[0].Data, ent.EntName)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(&ent, "")
		if err != nil {
			return err
		}
		hdr.Name = path.Join(curpath, ent.EntName)
		err = out.WriteHeader(hdr)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, f)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
