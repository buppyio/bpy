package mkdir

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/remote"
	"encoding/hex"
	"flag"
)

func Mkdir() {
	tagArg := flag.String("tag", "default", "tag make directory in")
	flag.Parse()

	if len(flag.Args()) != 1 {
		common.Die("please specify the directory to create\n")
	}

	destPath := flag.Args()[0]

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

	tagHash, ok, err := remote.GetTag(c, *tagArg)
	if err != nil {
		common.Die("error fetching tag hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("tag '%s' does not exist\n", *tagArg)
	}

	destHash, err := bpy.ParseHash(tagHash)
	if err != nil {
		common.Die("error parsing hash: %s\n", err.Error())
	}

	emptyDirEnt, err := fs.EmptyDir(store, 0755)
	if err != nil {
		common.Die("error copying data: %s\n", err.Error())
	}

	newRoot, err := fs.Insert(store, destHash, destPath, emptyDirEnt)
	if err != nil {
		common.Die("error inserting empty dir into folder: %s\n", err.Error())
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}

	ok, err = remote.CasTag(c, *tagArg, hex.EncodeToString(destHash[:]), hex.EncodeToString(newRoot.Data[:]))
	if err != nil {
		common.Die("creating tag: %s\n", err.Error())
	}

	if !ok {
		// XXX: loop here
		common.Die("tag concurrently modified, try again\n")
	}

}
