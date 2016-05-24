package ls

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/tags"
	"flag"
	"fmt"
)

func Ls() {
	var root [32]byte
	tagArg := flag.String("tag", "", "tag of directory to list")
	hashArg := flag.String("hash", "", "hash of directory to list")
	flag.Parse()

	if *hashArg == "" && *tagArg == "" || *hashArg != "" && *tagArg != "" {
		common.Die("please specify a hash or a tag to list\n")
	}

	if *hashArg != "" {
		hash, err := bpy.ParseHash(*hashArg)
		if err != nil {
			common.Die("error parsing hash: %s\n", err.Error())
		}
		root = hash
	}

	if len(flag.Args()) != 1 {
		common.Die("please specify a path\n")
	}

	remote, err := common.GetRemote()
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}

	store, err := common.GetCStoreReader(remote)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	if *tagArg != "" {
		tagHash, err := tags.Get(remote, *tagArg)
		if err != nil {
			common.Die("error fetching tag hash: %s\n", err.Error())
		}
		root, err = bpy.ParseHash(tagHash)
		if err != nil {
			common.Die("error parsing hash: %s\n", err.Error())
		}
	}

	ents, err := fs.Ls(store, root, flag.Args()[0])
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
		common.Die("error closing content store: %s\n", err.Error())
	}
}
