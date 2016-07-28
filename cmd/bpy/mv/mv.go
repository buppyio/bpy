package cp

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/fs/fsutil"
	"acha.ninja/bpy/remote"
	"encoding/hex"
	"flag"
	"path"
	"path/filepath"
	"strings"
)

func Mv() {
	tagArg := flag.String("tag", "default", "tag put data into")
	flag.Parse()

	if len(flag.Args()) != 2 {
		common.Die("please specify a src and dest\n")
	}
	srcPath := flag.Args()[0]
	destPath := flag.Args()[1]

	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()
	wstore, err := common.GetCStoreWriter(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	rstore, err := common.GetCStoreReader(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	tagVal, ok, err := remote.GetTag(c, *tagArg)
	if err != nil {
		common.Die("error fetching tag hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("tag '%s' does not exist\n", *tagArg)
	}

	rootHash, err := bpy.ParseHash(tagVal)
	if err != nil {
		common.Die("error parsing hash: %s\n", err.Error())
	}

	newRoot, err := fs.Move(rstore, wstore, rootHash, destPath, srcPath)
	if err != nil {
		common.Die("error moving folder: %s\n", err.Error())
	}

	err = wstore.Close()
	if err != nil {
		common.Die("error closing wstore: %s\n", err.Error())
	}

	err = rstore.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}

	ok, err = remote.CasTag(c, *tagArg, tagVal, hex.EncodeToString(newRoot.Data[:]))
	if err != nil {
		common.Die("creating tag: %s\n", err.Error())
	}

	if !ok {
		// XXX: loop here
		common.Die("tag concurrently modified, try again\n")
	}

}
