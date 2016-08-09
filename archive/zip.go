package archive

import (
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/fs"
	"archive/zip"
	"io"
	"path"
)

func Zip(store bpy.CStore, dirHash [32]byte, out io.Writer) error {
	zw := zip.NewWriter(out)
	err := writeZip(store, "", dirHash, zw)
	if err != nil {
		return err
	}
	return zw.Close()
}

func writeZip(store bpy.CStore, curpath string, dirHash [32]byte, out *zip.Writer) error {
	ents, err := fs.ReadDir(store, dirHash)
	if err != nil {
		return err
	}
	for _, ent := range ents[1:] {
		if ent.IsDir() {
			err = writeZip(store, path.Join(curpath, ent.EntName), ent.Data, out)
			if err != nil {
				return err
			}
			continue
		}
		f, err := fs.Open(store, ents[0].Data, ent.EntName)
		if err != nil {
			return err
		}
		defer f.Close()
		hdr, err := zip.FileInfoHeader(&ent)
		if err != nil {
			return err
		}
		hdr.Name = path.Join(curpath, ent.EntName)
		outfile, err := out.CreateHeader(hdr)
		if err != nil {
			return err
		}
		_, err = io.Copy(outfile, f)
		if err != nil {
			return err
		}
	}
	return nil
}
