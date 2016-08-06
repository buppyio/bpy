package get

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs/fsutil"
	"acha.ninja/bpy/remote"
	"flag"
)

func Get() {
	var root [32]byte
	tagArg := flag.String("tag", "default", "tag of directory to list")
	pathArg := flag.String("path", "", "directory to get")
	flag.Parse()

	if len(flag.Args()) != 1 {
		common.Die("please specify a dest path\n")
	}

	if *pathArg == "" {
		common.Die("please specify a path to fetch with a -path argument\n")
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
		common.Die("tag '%s' does not exist", *tagArg)
	}

	root, err = bpy.ParseHash(tagHash)
	if err != nil {
		common.Die("error parsing hash: %s\n", err.Error())
	}

	err = fsutil.CpFsToHost(store, root, *pathArg, flag.Args()[0])
	if err != nil {
		common.Die("error copying directory: %s\n", err.Error())
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}
}
