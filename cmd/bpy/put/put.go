package put

import (
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs/fsutil"
	"encoding/hex"
	"fmt"
	"os"
)

func Put() {
	store, err := common.GetCStoreWriter()
	if err != nil {
		panic(err)
	}
	hash, err := fsutil.CpHostDirToFs(store, os.Args[2])
	if err != nil {
		panic(err)
	}
	_, err = fmt.Println(hex.EncodeToString(hash[:]))
	if err != nil {
		panic(err)
	}
	err = store.Close()
	if err != nil {
		panic(err)
	}
}
