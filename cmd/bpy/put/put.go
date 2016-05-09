package put

import (
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs/fsutil"
	"encoding/hex"
	"flag"
	"fmt"
)

func Put() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		common.Die("please specify the dir to put\n")
	}
	store, err := common.GetCStoreWriter()
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	hash, err := fsutil.CpHostDirToFs(store, flag.Args()[0])
	if err != nil {
		common.Die("error copying data: %s\n", err.Error())
	}
	err = store.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}
	_, err = fmt.Println(hex.EncodeToString(hash[:]))
	if err != nil {
		common.Die("error printing hash: %s\n", err.Error())
	}
}
