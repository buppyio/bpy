package main

import (
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/fs/fsutil"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

type rstore struct {
	pack *bpack.Reader
}

func (s *rstore) Put(v []byte) ([32]byte, error) {
	panic("unimplemented")
}

func (s *rstore) Get(hash [32]byte) ([]byte, error) {
	data, ok, err := s.pack.Get(string(hash[:]))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%s not found", hex.EncodeToString(hash[:]))
	}
	return data, nil
}

type wstore struct {
	pack *bpack.Writer
}

func (s *wstore) Put(v []byte) ([32]byte, error) {
	hash := sha256.Sum256(v)
	return hash, s.pack.Add(string(hash[:]), v)
}

func (s *wstore) Get(hash [32]byte) ([]byte, error) {
	panic("unimplemented")
}

func put() {
	f, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
	}
	w, err := bpack.NewWriter(f)
	if err != nil {
		panic(err)
	}
	store := &wstore{pack: w}
	hash, err := fsutil.CpHostDirToFs(store, os.Args[3])
	if err != nil {
		panic(err)
	}
	err = w.Close()
	if err != nil {
		panic(err)
	}
	err = f.Close()
	if err != nil {
		panic(err)
	}
	_, err = fmt.Println(hex.EncodeToString(hash[:]))
	if err != nil {
		panic(err)
	}
}

func get() {
	var hash [32]byte
	f, err := os.Open(os.Args[2])
	if err != nil {
		panic(err)
	}
	r := bpack.NewReader(f)
	store := &rstore{pack: r}
	hbytes, err := hex.DecodeString(os.Args[3])
	if err != nil {
		panic(err)
	}
	copy(hash[:], hbytes)
	err = fsutil.CpFsDirToHost(store, hash, os.Args[4], 0666)
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
	}
}
