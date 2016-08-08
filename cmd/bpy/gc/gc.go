package gc

import (
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/gc"
	"acha.ninja/bpy/remote"
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
