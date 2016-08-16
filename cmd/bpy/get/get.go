package get

import (
	"flag"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs/fsutil"
	"github.com/buppyio/bpy/remote"
)

func Get() {
	var root [32]byte
	refArg := flag.String("ref", "default", "ref of directory to list")
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

	refHash, ok, err := remote.GetRef(c, *refArg)
	if err != nil {
		common.Die("error fetching ref hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("ref '%s' does not exist", *refArg)
	}

	root, err = bpy.ParseHash(refHash)
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
