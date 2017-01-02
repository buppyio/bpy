package revert

import (
	"flag"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"time"
)

func Revert() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		common.Die("please specify the hash to revert to\n")
	}

	revertToHash, err := bpy.ParseHash(flag.Args()[0])
	if err != nil {
		common.Die("error parsing hash: %s\n", err)
	}

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

	epoch, err := remote.GetEpoch(c)
	if err != nil {
		common.Die("error getting current epoch: %s\n", err.Error())
	}

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

	revertToRef, err := refs.GetRef(store, revertToHash)
	if err != nil {
		common.Die("error fetching ref: %s\n", err.Error())
	}

	revertToRef.Prev = rootHash
	revertToRef.HasPrev = true
	revertToRef.CreatedAt = time.Now().Unix()

	newRefHash, err := refs.PutRef(store, revertToRef)
	if err != nil {
		common.Die("error writing updated ref: %s\n", err.Error())
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing content store: %s\n", err.Error())
	}

	ok, err = remote.CasRoot(c, &k, newRefHash, bpy.NextRootVersion(rootVersion), epoch)
	if err != nil {
		common.Die("error swapping root: %s\n", err.Error())
	}
	if !ok {
		common.Die("root concurrently modified, try again\n")
	}
}
