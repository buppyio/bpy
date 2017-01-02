package gc

import (
	"flag"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/gc"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
)

func GC() {
	cancel := flag.Bool("cancel", false, "cancel any active garbage collection without starting a new one")
	keepHistory := flag.Bool("keep-history", false, "do not clear the gc history")

	flag.Parse()

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

	store, err := common.GetCStore(cfg, &k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	// Stop any gc that is currently running
	err = remote.StopGC(c)
	if err != nil {
		common.Die("error stopping gc: %s\n", err.Error())
	}
	if *cancel {
		return
	}

	if !*keepHistory {
		epoch, err := remote.GetEpoch(c)
		if err != nil {
			common.Die("error getting current epoch: %s\n", err.Error())
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

		newRef := ref
		newRef.HasPrev = false

		newRefHash, err := refs.PutRef(store, newRef)
		ok, err = remote.CasRoot(c, &k, newRefHash, bpy.NextRootVersion(rootVersion), epoch)
		if err != nil {
			common.Die("error swapping root: %s\n", err.Error())
		}
		if !ok {
			common.Die("root concurrently modified, try again\n")
		}

		err = store.Flush()
		if err != nil {
			common.Die("error flushing content store: %s\n", err.Error())
		}
	}

	cache, err := common.GetCacheClient(cfg)
	if err != nil {
		common.Die("error getting cache connection: %s\n", err.Error())
	}

	err = gc.GC(c, store, cache, &k)
	if err != nil {
		common.Die("error running gc: %s\n", err.Error())
	}
}
