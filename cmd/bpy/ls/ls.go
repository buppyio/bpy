package ls

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs"
	"fmt"
	"os"
)

func Ls() {
	hash, err := bpy.ParseHash(os.Args[2])
	if err != nil {
		panic(err)
	}
	store, err := common.GetCStoreReader()
	if err != nil {
		panic(err)
	}
	ents, err := fs.Ls(store, hash, os.Args[3])
	if err != nil {
		panic(err)
	}
	for _, ent := range ents[1:] {
		if ent.Mode.IsDir() {
			_, err = fmt.Printf("%s/\n", ent.Name)
		}
	}
	for _, ent := range ents[1:] {
		if !ent.Mode.IsDir() {
			_, err = fmt.Printf("%s\n", ent.Name)
		}
	}
	err = store.Close()
	if err != nil {
		panic(err)
	}
}
