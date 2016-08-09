package gc

import (
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/cstore"
	"github.com/buppyio/bpy/gc"
	"github.com/buppyio/bpy/remote"
)

func GC() {

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
	store = cstore.NewMemCachedCStore(store, 32*1024*1024)

	// Stop any gc that is currently running
	err = remote.StopGC(c)
	if err != nil {
		common.Die("error stopping gc: %s\n", err.Error())
	}

	err = gc.GC(c, store, &k)
	if err != nil {
		common.Die("error running gc: %s\n", err.Error())
	}
}
