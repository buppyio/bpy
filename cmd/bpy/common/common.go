package common

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/proto9"
	"net"
	"os/user"
	"path/filepath"
)

func GetBuppyDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, ".bpy"), nil
}

func GetCacheDir() (string, error) {
	d, err := GetBuppyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "cache"), nil
}

func GetStore() (*client9.Client, error) {
	con, err := net.Dial("tcp", "localhost:9001")
	if err != nil {
		return nil, err
	}
	store, err := client9.NewClient(proto9.NewConn(con, con, 65536))
	if err != nil {
		con.Close()
		return nil, err
	}
	err = store.Attach("ac", "")
	if err != nil {
		con.Close()
		return nil, err
	}
	return store, nil
}

func GetCStoreReader() (bpy.CStoreReader, error) {
	store, err := GetStore()
	if err != nil {
		return nil, err
	}
	cache, err := GetCacheDir()
	if err != nil {
		return nil, err
	}
	return cstore.NewReader(store, cache)
}

func GetCStoreWriter() (bpy.CStoreWriter, error) {
	store, err := GetStore()
	if err != nil {
		return nil, err
	}
	cache, err := GetCacheDir()
	if err != nil {
		return nil, err
	}
	return cstore.NewWriter(store, cache)
}
