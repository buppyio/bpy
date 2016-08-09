package newkey

import (
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"flag"
	"fmt"
	"os"
)

func NewKey() {

	keyFile := flag.String("f", "", "file to write the key to. (defaults to stdout)")

	flag.Parse()

	if *keyFile == "" {

	}

	f, err := os.OpenFile(*keyFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
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
