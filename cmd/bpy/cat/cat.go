package cat

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/remote"
	"flag"
	"io"
	"os"
)

func Cat() {
	var root [32]byte
	tagArg := flag.String("tag", "default", "tag of directory to list")
	flag.Parse()

	if len(flag.Args()) != 1 {
		common.Die("please specify a path\n")
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

	tagHash, ok, err := remote.GetTag(c, *tagArg)
	if err != nil {
		common.Die("error fetching tag hash: %s\n", err.Error())
	}

	if !ok {
		common.Die("tag '%s' does not exist\n", *tagArg)
	}

	root, err = bpy.ParseHash(tagHash)
	if err != nil {
		common.Die("error parsing hash: %s\n", err.Error())
	}

	for _, fpath := range flag.Args() {
		rdr, err := fs.Open(store, root, fpath)
		if err != nil {
			common.Die("error opening %s: %s\n", fpath, err.Error())
		}
		_, err = io.Copy(os.Stdout, rdr)
		if err != nil {
			common.Die("io error: %s\n", err.Error())
		}
	}
	err = store.Close()
	if err != nil {
		common.Die("error closing store: %s\n", err.Error())
	}
}
