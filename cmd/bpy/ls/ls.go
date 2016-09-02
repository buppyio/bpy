package ls

import (
	"flag"
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
)

func Ls() {
	refArg := flag.String("ref", "default", "ref of directory to list")
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

	refHash, ok, err := remote.GetNamedRef(c, &k, *refArg)
	if err != nil {
		common.Die("error fetching ref hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("ref '%s' does not exist", *refArg)
	}

	ref, err := refs.GetRef(store, refHash)
	if err != nil {
		common.Die("error fetching ref: %s\n", err.Error())
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
