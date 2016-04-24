package common

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cstore"
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

func GetStoreDir() (string, error) {
	d, err := GetBuppyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "store"), nil
}

func GetCacheDir() (string, error) {
	d, err := GetBuppyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "cache"), nil
}

func GetCStoreReader() (bpy.CStoreReader, error) {
	store, err := GetStoreDir()
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
	store, err := GetStoreDir()
	if err != nil {
		return nil, err
	}
	cache, err := GetCacheDir()
	if err != nil {
		return nil, err
	}
	return cstore.NewWriter(store, cache)
}
