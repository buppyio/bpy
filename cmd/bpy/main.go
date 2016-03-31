package main

import (
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/fs/fsutil"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func put() {
	store, err := cstore.NewWriter("/home/ac/.bpy/store", "/home/ac/.bpy/cache")
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

func get() {
	var hash [32]byte
	store, err := cstore.NewReader("/home/ac/.bpy/store", "/home/ac/.bpy/cache")
	if err != nil {
		panic(err)
	}
	hbytes, err := hex.DecodeString(os.Args[2])
	if err != nil {
		panic(err)
	}
	copy(hash[:], hbytes)
	err = fsutil.CpFsDirToHost(store, hash, os.Args[3])
	if err != nil {
		panic(err)
	}
	err = store.Close()
	if err != nil {
		panic(err)
	}
}

func ls() {
	var hash [32]byte
	store, err := cstore.NewReader("/home/ac/.bpy/store", "/home/ac/.bpy/cache")
	if err != nil {
		panic(err)
	}
	hbytes, err := hex.DecodeString(os.Args[2])
	if err != nil {
		panic(err)
	}
	copy(hash[:], hbytes)
	ents, err := fs.Ls(store, hash, os.Args[3])
	if err != nil {
		panic(err)
	}
	for _, ent := range ents[1:] {
		if ent.Mode.IsDir() {
			_, err = fmt.Printf("%s/\n", ent.Name)
		} else {
			_, err = fmt.Printf("%s\n", ent.Name)
		}
		if err != nil {
			panic(err)
		}
	}
	err = store.Close()
	if err != nil {
		panic(err)
	}
}

func cat() {
	var hash [32]byte
	store, err := cstore.NewReader("/home/ac/.bpy/store", "/home/ac/.bpy/cache")
	if err != nil {
		panic(err)
	}
	hbytes, err := hex.DecodeString(os.Args[2])
	if err != nil {
		panic(err)
	}
	copy(hash[:], hbytes)

	for _, fpath := range os.Args[3:] {
		rdr, err := fs.Open(store, hash, fpath)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(os.Stdout, rdr)
		if err != nil {
			panic(err)
		}
	}
	err = store.Close()
	if err != nil {
		panic(err)
	}
}

func main() {
	switch os.Args[1] {
	case "put":
		put()
	case "get":
		get()
	case "cat":
		cat()
	case "ls":
		ls()
	case "dbg":
		dbg()
	}
}
