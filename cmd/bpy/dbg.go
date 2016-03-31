package main

import (
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/htree"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func inspecthtree() {
	var hash [32]byte
	store, err := cstore.NewReader("/home/ac/.bpy/store", "/home/ac/.bpy/cache")
	if err != nil {
		panic(err)
	}
	defer store.Close()
	hbytes, err := hex.DecodeString(os.Args[3])
	if err != nil {
		panic(err)
	}
	copy(hash[:], hbytes)

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
	store, err := cstore.NewWriter("/home/ac/.bpy/store", "/home/ac/.bpy/cache")
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
