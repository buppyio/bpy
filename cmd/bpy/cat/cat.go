package cat

import (
	"flag"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/remote"
	"io"
	"os"
)

func Cat() {
	var root [32]byte
	refArg := flag.String("ref", "default", "ref of directory to list")
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

	refHash, ok, err := remote.GetRef(c, *refArg)
	if err != nil {
		common.Die("error fetching ref hash: %s\n", err.Error())
	}

	if !ok {
		common.Die("ref '%s' does not exist\n", *refArg)
	}

	root, err = bpy.ParseHash(refHash)
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
