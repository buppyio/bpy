package mv

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"time"
)

func Mv() {
	flag.Parse()

	if len(flag.Args()) != 2 {
		common.Die("please specify a src and dest\n")
	}
	srcPath := flag.Args()[0]
	destPath := flag.Args()[1]

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

		newRoot, err := fs.Move(store, ref.Root, destPath, srcPath)
		if err != nil {
			common.Die("error moving folder: %s\n", err.Error())
		}

		newRefHash, err := refs.PutRef(store, refs.Ref{
			CreatedAt: time.Now().Unix(),
			Root:      newRoot.HTree.Data,
			HasPrev:   true,
			Prev:      refHash,
		})
		if err != nil {
			common.Die("error creating new ref: %s\n", err.Error())
		}

		err = store.Close()
		if err != nil {
			common.Die("error closing remote: %s\n", err.Error())
		}

		ok, err = remote.CasRef(c, &k, refHash, newRefHash, generation)
		if err != nil {
			common.Die("creating ref: %s\n", err.Error())
		}

		if ok {
			break
		}
	}

}
