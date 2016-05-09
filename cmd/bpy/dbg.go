package main

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/htree"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
)

func dbghelp() {
	fmt.Println("Please specify one of the following subcommands:")
	fmt.Println("inspect-htree, write-htree\n")
	os.Exit(1)
}

func inspecthtree() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		common.Die("please specify a single hash\n")
	}
	hash, err := bpy.ParseHash(flag.Args()[0])
	if err != nil {
		common.Die("error parsing hash: %s\n", err.Error())
	}
	store, err := common.GetCStoreReader()
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	data, err := store.Get(hash)
	if err != nil {
		common.Die("error getting hash: %s", err.Error())
	}
	_, err = fmt.Printf("level: %d\n", int(data[0]))
	if err != nil {
		common.Die("io error: %s\n", err.Error())
	}
	if data[0] == 0 {
		return
	}
	data = data[1:]
	for len(data) != 0 {
		offset := binary.LittleEndian.Uint64(data[0:8])
		hashstr := hex.EncodeToString(data[8:40])
		_, err := fmt.Printf("%d %s\n", offset, hashstr)
		if err != nil {
			common.Die("io error: %s\n", err.Error())
		}
		data = data[40:]
	}
}

func writehtree() {
	store, err := common.GetCStoreWriter()
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	w := htree.NewWriter(store)
	_, err = io.Copy(w, os.Stdin)
	if err != nil {
		common.Die("io error: %s\n", err.Error())
	}
	h, err := w.Close()
	if err != nil {
		common.Die("error closing htree: %s\n", err.Error())
	}
	err = store.Close()
	if err != nil {
		common.Die("error closing connection: %s\n", err.Error())
	}
	_, err = fmt.Printf("%s\n", hex.EncodeToString(h[:]))
	if err != nil {
		common.Die("error printing hash: %s\n", err.Error())
	}
}

func dbg() {
	cmd := dbghelp
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "inspect-htree":
			cmd = inspecthtree
		case "write-htree":
			cmd = writehtree
		default:
			cmd = dbghelp
		}
		copy(os.Args[1:], os.Args[2:])
		os.Args = os.Args[0 : len(os.Args)-1]
	}
	cmd()
}
