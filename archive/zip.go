package archive

import (
	"archive/zip"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/fs"
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
			err = writeZip(store, path.Join(curpath, ent.EntName), ent.HTree.Data, out)
			if err != nil {
				return err
			}
			continue
		}
		f, err := fs.Open(store, ents[0].HTree.Data, ent.EntName)
		if err != nil {
			return err
		}
		defer f.Close()
		hdr, err := zip.FileInfoHeader(&ent)
		if err != nil {
			return err
		}
		hdr.Name = path.Join(curpath, ent.EntName)
		hdr.Method = zip.Deflate
		outFile, err := out.CreateHeader(hdr)
		if err != nil {
			return err
		}
		_, err = io.Copy(outFile, f)
		if err != nil {
			return err
		}
	}
	return nil
}
