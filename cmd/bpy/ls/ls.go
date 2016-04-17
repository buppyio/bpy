package ls

import (
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/fs"
	"encoding/hex"
	"fmt"
	"os"
)

func Ls() {
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
	ents, err := fs.Ls(store, hash, os.Args[3])
	if err != nil {
		panic(err)
	}
	for _, ent := range ents[1:] {
		if ent.Mode.IsDir() {
			_, err = fmt.Printf("%s/\n", ent.Name)
		} else {
			_, err = fmt.Printf("%s\n", ent.Name)
		}
		if err != nil {
			panic(err)
		}
	}
	err = store.Close()
	if err != nil {
		panic(err)
	}
}
