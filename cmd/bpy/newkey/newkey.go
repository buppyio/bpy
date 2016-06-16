package newkey

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"flag"
	"fmt"
	"io"
	"os"
)

func NewKey() {
	var o io.WriteCloser

	keyFile := flag.String("f", "", "file to write the key to. (defaults to stdout)")

	flag.Parse()

	if *keyFile == "" {
		o = os.Stdout
	} else {
		f, err := os.OpenFile(*keyFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			common.Die("error creating key file: %s", err.Error())
		}
		o = f
	}

	k, err := bpy.NewKey()
	if err != nil {
		common.Die("%s", err.Error())
	}

	err = bpy.WriteKey(o, &k)
	if err != nil {
		common.Die("%s", err.Error())
	}

	_, err = fmt.Fprintln(o, "")
	if err != nil {
		common.Die("%s", err.Error())
	}
}
