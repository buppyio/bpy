package put

import (
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs/fsutil"
	"acha.ninja/bpy/tags"
	"encoding/hex"
	"flag"
	"fmt"
)

func Put() {
	tagArg := flag.String("tag", "", "create tag")
	flag.Parse()

	if len(flag.Args()) != 1 {
		common.Die("please specify the dir to put\n")
	}

	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	remote, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}

	store, err := common.GetCStoreWriter(&k, remote)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
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

	if *tagArg != "" {
		err = tags.Create(remote, *tagArg, hex.EncodeToString(hash[:]))
		if err != nil {
			common.Die("error creating tag: %s\n", err.Error())
		}
	}

}
