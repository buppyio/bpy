package ls

import (
	"flag"
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/when"
)

func Ls() {
	whenArg := flag.String("when", "", "time query")

	lsPath := "/"
	flag.Parse()

	if len(flag.Args()) > 1 {
		common.Die("please specify a single path\n")
	} else if len(flag.Args()) == 1 {
		lsPath = flag.Args()[0]
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

	rootHash, ok, err := remote.GetRoot(c, &k)
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

	ents, err := fs.Ls(store, ref.Root, lsPath)
	if err != nil {
		common.Die("error reading directory: %s\n", err.Error())
	}

	for _, ent := range ents[1:] {
		if ent.EntMode.IsDir() {
			_, err = fmt.Printf("%s/\n", ent.EntName)
			if err != nil {
				common.Die("io error: %s\n", err.Error())
			}
		}
	}
	for _, ent := range ents[1:] {
		if !ent.EntMode.IsDir() {
			_, err = fmt.Printf("%s\n", ent.EntName)
			if err != nil {
				common.Die("io error: %s\n", err.Error())
			}
		}
	}
	err = store.Close()
	if err != nil {
		common.Die("error closing content store: %s\n", err.Error())
	}
}
