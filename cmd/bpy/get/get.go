package get

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs/fsutil"
	"flag"
)

func Get() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		common.Die("please specify the hash to get and the destination directory\n")
	}
	hash, err := bpy.ParseHash(flag.Args()[0])
	if err != nil {
		common.Die("error parsing given hash: %s\n", err.Error())
	}
	store, err := common.GetCStoreReader()
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	err = fsutil.CpFsDirToHost(store, hash, flag.Args()[1])
	if err != nil {
		common.Die("error copying directory: %s\n", err.Error())
	}
	err = store.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}
}
