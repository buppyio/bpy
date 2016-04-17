package get

import (
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/fs/fsutil"
	"encoding/hex"
	"os"
)

func Get() {
	var hash [32]byte
	store, err := cstore.NewReader("/home/ac/.bpy/store", "/home/ac/.bpy/cache")
	if err != nil {
		panic(err)
	}
	hbytes, err := hex.DecodeString(os.Args[2])
	if err != nil {
		panic(err)
	}
	copy(hash[:], hbytes)
	err = fsutil.CpFsDirToHost(store, hash, os.Args[3])
	if err != nil {
		panic(err)
	}
	err = store.Close()
	if err != nil {
		panic(err)
	}
}
