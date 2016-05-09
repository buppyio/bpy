package ls

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs"
	"flag"
	"fmt"
)

func Ls() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		common.Die("please specify a directory hash and path\n")
	}
	hash, err := bpy.ParseHash(flag.Args()[0])
	if err != nil {
		common.Die("error parsing root hash: %s\n", err.Error())
	}
	store, err := common.GetCStoreReader()
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	ents, err := fs.Ls(store, hash, flag.Args()[1])
	if err != nil {
		common.Die("error reading directory: %s\n", err.Error())
	}
	for _, ent := range ents[1:] {
		if ent.Mode.IsDir() {
			_, err = fmt.Printf("%s/\n", ent.Name)
			if err != nil {
				common.Die("io error: %s\n", err.Error())
			}
		}
	}
	for _, ent := range ents[1:] {
		if !ent.Mode.IsDir() {
			_, err = fmt.Printf("%s\n", ent.Name)
			if err != nil {
				common.Die("io error: %s\n", err.Error())
			}
		}
	}
	err = store.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}
}
