package gc

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/gc"
	"github.com/buppyio/bpy/remote"
)

func GC() {
	cancel := flag.Bool("cancel", false, "cancel any active garbage collection without starting a new one")

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

	cache, err := common.GetCacheClient(cfg)
	if err != nil {
		common.Die("error getting cache connection: %s\n", err.Error())
	}

	err = gc.GC(c, store, cache, &k)
	if err != nil {
		common.Die("error running gc: %s\n", err.Error())
	}
}
