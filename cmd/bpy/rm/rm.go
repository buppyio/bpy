package rm

import (
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/remote"
	"encoding/hex"
	"flag"
)

func Rm() {
	tagArg := flag.String("tag", "default", "tag put rm from")
	flag.Parse()

	if len(flag.Args()) != 1 {
		common.Die("please path to remove\n")
	}

	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	store, err := common.GetCStore(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	generation, err := remote.GetGeneration(c)
	if err != nil {
		common.Die("error getting current gc generation: %s\n", err.Error())
	}

	tagHash, ok, err := remote.GetTag(c, *tagArg)
	if err != nil {
		common.Die("error fetching tag hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("tag '%s' does not exist\n", *tagArg)
	}

	rootHash, err := bpy.ParseHash(tagHash)
	if err != nil {
		common.Die("error parsing hash: %s\n", err.Error())
	}

	newRoot, err := fs.Remove(store, rootHash, flag.Args()[0])
	if err != nil {
		common.Die("error removing file: %s\n", err.Error())
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing store: %s\n", err.Error())
	}

	ok, err = remote.CasTag(c, *tagArg, hex.EncodeToString(rootHash[:]), hex.EncodeToString(newRoot.Data[:]), generation)
	if err != nil {
		common.Die("creating tag: %s\n", err.Error())
	}

	if !ok {
		// XXX: loop here
		common.Die("tag concurrently modified, try again\n")
	}

}
