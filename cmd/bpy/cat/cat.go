package cat

import (
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/fs"
	"encoding/hex"
	"io"
	"os"
)

func Cat() {
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
