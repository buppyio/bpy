package zip

import (
	"flag"
	"github.com/buppyio/bpy/archive"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/remote"
	"os"
)

func Zip() {
	refArg := flag.String("ref", "default", "ref of directory to list")
	srcArg := flag.String("src", "", "path to directory to ref")
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

	ref, ok, err := remote.GetRef(c, &k, *refArg)
	if err != nil {
		common.Die("error fetching ref hash: %s\n", err.Error())
	}

	if !ok {
		common.Die("ref '%s' does not exist\n", *refArg)
	}

	dirEnt, err := fs.Walk(store, ref.Root, *srcArg)
	if err != nil {
		common.Die("error getting src folder: %s\n", err.Error())
	}
	if !dirEnt.IsDir() {
		common.Die("'%s' is not a directory\n", *srcArg)
	}

	err = archive.Zip(store, dirEnt.HTree.Data, os.Stdout)
	if err != nil {
		common.Die("error writing zip: %s\n", err.Error())
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing store: %s\n", err.Error())
	}
}
