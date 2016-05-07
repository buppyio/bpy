package cat

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs"
	"flag"
	"io"
	"os"
)

func Cat() {
	flag.Parse()
	if len(flag.Args()) < 3 {
		common.Die("please specify the hash to get and the destination directory\n")
	}
	hash, err := bpy.ParseHash(flag.Args()[1])
	if err != nil {
		common.Die("error parsing given hash: %s\n", err.Error())
	}
	store, err := common.GetCStoreReader()
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	for _, fpath := range flag.Args()[2:] {
		rdr, err := fs.Open(store, hash, fpath)
		if err != nil {
			common.Die("error opening %s: %s\n", fpath, err.Error())
		}
		_, err = io.Copy(os.Stdout, rdr)
		if err != nil {
			common.Die("io error: %s\n", err.Error())
		}
	}
	err = store.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}
}
