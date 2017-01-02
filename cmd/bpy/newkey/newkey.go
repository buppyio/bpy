package newkey

import (
	"flag"
	"fmt"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"os"
)

func NewKey() {
	keyFile := flag.String("f", "", "file to write the key to. (defaults to $BPY_PATH/bpy.key)")

	flag.Parse()

	if *keyFile == "" {
		cfg, err := common.GetConfig()
		if err != nil {
			common.Die("error getting config: %s\n", err)
		}
		*keyFile = cfg.KeyPath
	}

	f, err := os.OpenFile(*keyFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0400)
	if err != nil {
		common.Die("error creating key file: %s\n", err.Error())
	}

	k, err := bpy.NewKey()
	if err != nil {
		common.Die("%s\n", err.Error())
	}

	err = bpy.WriteKey(f, &k)
	if err != nil {
		common.Die("%s\n", err.Error())
	}

	_, err = fmt.Fprintln(f, "")
	if err != nil {
		common.Die("%s\n", err.Error())
	}

	err = f.Close()
	if err != nil {
		common.Die("%s\n", err.Error())
	}

}
