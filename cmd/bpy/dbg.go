package main

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/htree"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func inspecthtree() {
	store, err := common.GetCStoreReader()
	if err != nil {
		panic(err)
	}
	hash, err := bpy.ParseHash(os.Args[3])
	if err != nil {
		panic(err)
	}
	data, err := store.Get(hash)
	if err != nil {
		panic(err)
	}
	_, err = fmt.Printf("level: %d\n", int(data[0]))
	if err != nil {
		panic(err)
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
			panic(err)
		}
		data = data[40:]
	}
}

func writehtree() {
	store, err := common.GetCStoreWriter()
	if err != nil {
		panic(err)
	}
	w := htree.NewWriter(store)
	_, err = io.Copy(w, os.Stdin)
	if err != nil {
		panic(err)
	}
	h, err := w.Close()
	if err != nil {
		panic(err)
	}
	err = store.Close()
	if err != nil {
		panic(err)
	}
	_, err = fmt.Printf("%s\n", hex.EncodeToString(h[:]))
	if err != nil {
		panic(err)
	}
}

func dbg() {
	switch os.Args[2] {
	case "inspect-htree":
		inspecthtree()
	case "write-htree":
		writehtree()
	}
}
