package cat

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs"
	"io"
	"os"
)

func Cat() {
	hash, err := bpy.ParseHash(os.Args[2])
	if err != nil {
		panic(err)
	}
	store, err := common.GetCStoreReader()
	if err != nil {
		panic(err)
	}
	for _, fpath := range os.Args[3:] {
		rdr, err := fs.Open(store, hash, fpath)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(os.Stdout, rdr)
		if err != nil {
			panic(err)
		}
	}
	err = store.Close()
	if err != nil {
		panic(err)
	}
}
