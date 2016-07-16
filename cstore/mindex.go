package cstore

import (
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/remote"
	"acha.ninja/bpy/remote/client"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path"
	"path/filepath"
)

func searchMetaIndex(midx []metaIndexEnt, hash [32]byte) (metaIndexEnt, bpack.IndexEnt, bool) {
	k := string(hash[:])
	for i := range midx {
		packidx, ok := midx[i].idx.Search(k)
		if ok {
			return midx[i], midx[i].idx[packidx], true
		}
	}
	return metaIndexEnt{}, bpack.IndexEnt{}, false
}

func readAndCacheMetaIndex(store *client.Client, key [32]byte, cachepath string) ([]metaIndexEnt, error) {
	listing, err := remote.ListPacks(store)
	if err != nil {
		return nil, err
	}
	midx := make([]metaIndexEnt, 0, 16)
	for _, pack := range listing {
		idx, err := getAndCacheIndex(store, key, pack.Name, pack.Size, cachepath)
		if err != nil {
			return nil, err
		}
		midxent := metaIndexEnt{
			packname: pack.Name,
			packsize: pack.Size,
			idx:      idx,
		}
		midx = append(midx, midxent)
	}
	return midx, nil
}

func randFileName() (string, error) {
	namebuf := [32]byte{}
	_, err := io.ReadFull(rand.Reader, namebuf[:])
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(namebuf[:]), nil
}

func cacheIndex(packname, cachepath string, index bpack.Index) error {
	idxpath := filepath.Join(cachepath, packname+".index")
	_, err := os.Stat(idxpath)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	tmpname, err := randFileName()
	if err != nil {
		return err
	}
	tmppath := filepath.Join(cachepath, tmpname)
	tmpf, err := os.Create(tmppath)
	if err != nil {
		return err
	}
	err = bpack.WriteIndex(tmpf, index)
	if err != nil {
		tmpf.Close()
		os.Remove(tmppath)
		return err
	}
	err = tmpf.Close()
	if err != nil {
		os.Remove(tmppath)
	}
	err = os.Rename(tmppath, idxpath)
	if err != nil {
		os.Remove(tmppath)
		return err
	}
	return nil
}

func getAndCacheIndex(store *client.Client, key [32]byte, packname string, packsize uint64, cachepath string) (bpack.Index, error) {
	idxpath := filepath.Join(cachepath, packname+".index")
	_, err := os.Stat(idxpath)
	if err == nil {
		f, err := os.Open(idxpath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return bpack.ReadIndex(f)
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	packPath := path.Join("packs", packname)
	f, err := store.Open(packPath)
	if err != nil {
		return nil, err
	}
	pack, err := bpack.NewEncryptedReader(f, key, int64(packsize))
	if err != nil {
		return nil, err
	}
	defer pack.Close()
	err = pack.ReadIndex()
	if err != nil {
		return nil, err
	}
	err = cacheIndex(packname, cachepath, pack.Idx)
	if err != nil {
		return nil, err
	}
	return pack.Idx, nil
}
