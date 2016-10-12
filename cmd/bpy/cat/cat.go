package cat

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/when"
	"io"
	"os"
)

func Cat() {
	whenArg := flag.String("when", "", "time query")

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

	_, rootHash, ok, err := remote.GetRoot(c, &k)
	if err != nil {
		common.Die("error fetching root hash: %s\n", err.Error())
	}

	if !ok {
		common.Die("root missing\n")
	}

	ref, err := refs.GetRef(store, rootHash)
	if err != nil {
		common.Die("error fetching ref: %s\n", err.Error())
	}

	if *whenArg != "" {
		refTime, err := when.Parse(*whenArg)
		if err != nil {
			common.Die("error parsing 'when' arg: %s\n", err.Error())
		}
		refPast, ok, err := refs.GetAtTime(store, ref, refTime)
		if err != nil {
			common.Die("error looking at ref history: %s\n", err.Error())
		}
		if !ok {
			common.Die("ref did not exist at %s\n", refTime.String())
		}
		ref = refPast
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
