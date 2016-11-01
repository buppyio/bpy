package cp

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/when"
	"time"
)

func Cp() {
	whenArg := flag.String("when", "", "time spec of the time to copy from")
	flag.Parse()

	if len(flag.Args()) != 2 {
		common.Die("please specify a src and dest\n")
	}
	srcPath := flag.Args()[0]
	destPath := flag.Args()[1]

	cfg, err := common.GetConfig()
	if err != nil {
		common.Die("error getting config: %s\n", err)
	}

	k, err := common.GetKey(cfg)
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(cfg, &k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	generation, err := remote.GetGeneration(c)
	if err != nil {
		common.Die("error getting current gc generation: %s\n", err.Error())
	}

	for {
		store, err := common.GetCStore(cfg, &k, c)
		if err != nil {
			common.Die("error getting content store: %s\n", err.Error())
		}

		rootHash, rootVersion, ok, err := remote.GetRoot(c, &k)
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

		newRoot, err := fs.Copy(store, ref.Root, destPath, srcPath)
		if err != nil {
			common.Die("error copying src to dest: %s\n", err.Error())
		}

		newRefHash, err := refs.PutRef(store, refs.Ref{
			CreatedAt: time.Now().Unix(),
			Root:      newRoot.HTree.Data,
			HasPrev:   true,
			Prev:      rootHash,
		})
		if err != nil {
			common.Die("error creating new ref: %s\n", err.Error())
		}

		err = store.Close()
		if err != nil {
			common.Die("error closing remote: %s\n", err.Error())
		}

		ok, err = remote.CasRoot(c, &k, newRefHash, rootVersion+1, generation)
		if err != nil {
			common.Die("error swapping root: %s\n", err.Error())
		}
		if ok {
			break
		}
	}

}
