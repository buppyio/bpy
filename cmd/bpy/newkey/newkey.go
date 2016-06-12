package newkey

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"crypto/rand"
	"encoding/json"
	"flag"
	"io"
	"os"
)

func NewKey() {
	var o io.WriteCloser
	var k bpy.Key

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

	_, err := io.ReadFull(rand.Reader, k.CipherKey[:])
	if err != nil {
		common.Die("error generating cipher key: %s", err.Error())
	}

	_, err = io.ReadFull(rand.Reader, k.HmacKey[:])
	if err != nil {
		common.Die("error generating hmac key: %s", err.Error())
	}

	_, err = io.ReadFull(rand.Reader, k.Id[:])
	if err != nil {
		common.Die("error generating id: %s", err.Error())
	}

	j, err := json.Marshal(&k)
	if err != nil {
		common.Die("error mashalling key: %s", err.Error())
	}

	_, err = o.Write(j)
	if err != nil {
		common.Die("error writing key: %s", err.Error())
	}

}
