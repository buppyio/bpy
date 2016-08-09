package tar

import (
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/archive"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/remote"
	"flag"
	"os"
)

func Tar() {
	var root [32]byte
	tagArg := flag.String("tag", "default", "tag of directory to list")
	srcArg := flag.String("src", "", "path to directory to tag")
	flag.Parse()

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

	dirEnt, err := fs.Walk(store, root, *srcArg)
	if err != nil {
		common.Die("error getting src folder: %s\n", err.Error())
	}
	if !dirEnt.IsDir() {
		common.Die("'%s' is not a directory", *srcArg)
	}

	err = archive.Tar(store, dirEnt.Data, os.Stdout)
	if err != nil {
		common.Die("error writing tar:", err)
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing store: %s\n", err.Error())
	}
}
