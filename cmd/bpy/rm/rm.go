package rm

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"time"
)

func Rm() {
	refArg := flag.String("ref", "default", "ref put rm from")
	flag.Parse()

	if len(flag.Args()) != 1 {
		common.Die("please path to remove\n")
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

	generation, err := remote.GetGeneration(c)
	if err != nil {
		common.Die("error getting current gc generation: %s\n", err.Error())
	}

	for {
		store, err := common.GetCStore(&k, c)
		if err != nil {
			common.Die("error getting content store: %s\n", err.Error())
		}

		refHash, ok, err := remote.GetNamedRef(c, &k, *refArg)
		if err != nil {
			common.Die("error fetching ref hash: %s\n", err.Error())
		}
		if !ok {
			common.Die("ref '%s' does not exist\n", *refArg)
		}

		ref, err := refs.GetRef(store, refHash)
		if err != nil {
			common.Die("error fetching ref: %s\n", err.Error())
		}

		newRootEnt, err := fs.Remove(store, ref.Root, flag.Args()[0])
		if err != nil {
			common.Die("error removing file: %s\n", err.Error())
		}

		newRefHash, err := refs.PutRef(store, refs.Ref{
			CreatedAt: time.Now().Unix(),
			Root:      newRootEnt.HTree.Data,
			HasPrev:   true,
			Prev:      refHash,
		})

		err = store.Close()
		if err != nil {
			common.Die("error closing store: %s\n", err.Error())
		}

		ok, err = remote.CasNamedRef(c, &k, *refArg, refHash, newRefHash, generation)
		if err != nil {
			common.Die("creating ref: %s\n", err.Error())
		}

		if ok {
			break
		}

	}
}
