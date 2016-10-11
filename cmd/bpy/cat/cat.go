package cat

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"io"
	"os"
)

func Cat() {
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

	refHash, ok, err := remote.GetRef(c, &k)
	if err != nil {
		common.Die("error fetching ref hash: %s\n", err.Error())
	}

	if !ok {
		common.Die("root missing\n")
	}

	ref, err := refs.GetRef(store, refHash)
	if err != nil {
		common.Die("error fetching ref: %s\n", err.Error())
	}

	for _, fpath := range flag.Args() {
		rdr, err := fs.Open(store, ref.Root, fpath)
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
