package get

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs/fsutil"
	"os"
)

func Get() {
	hash, err := bpy.ParseHash(os.Args[2])
	if err != nil {
		panic(err)
	}
	store, err := common.GetCStoreReader()
	if err != nil {
		panic(err)
	}
	err = fsutil.CpFsDirToHost(store, hash, os.Args[3])
	if err != nil {
		panic(err)
	}
	err = store.Close()
	if err != nil {
		panic(err)
	}
}
