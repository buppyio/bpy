package cstore

import (
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/bpack"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/remote/client"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type packInfo struct {
	Name string
	Size uint64
	Idx  bpack.Index
}

type metaIndex map[string]*packInfo

func searchMetaIndex(midx metaIndex, hash [32]byte) (*packInfo, bpack.IndexEnt, bool) {
	k := string(hash[:])
	info, ok := midx[k]
	if ok {
		packIdx, ok := info.Idx.Search(k)
		if !ok {
			panic("corrupt meta index")
		}
		return info, info.Idx[packIdx], true
	}
	return nil, bpack.IndexEnt{}, false
}

func cleanOldIndexes(packs []remote.PackListing, cachepath string) error {
	indexSet := make(map[string]struct{})
	for _, pack := range packs {
		indexSet[pack.Name+".index"] = struct{}{}
	}

	dirEnts, err := ioutil.ReadDir(cachepath)
	if err != nil {
		return err
	}

	for _, ent := range dirEnts {
		if !strings.HasSuffix(ent.Name(), ".index") {
			continue
		}
		_, ok := indexSet[ent.Name()]
		if ok {
			continue
		}
		fullPath := filepath.Join(cachepath, ent.Name())
		_ = os.Remove(fullPath)
	}
	return nil
}

func readAndCacheMetaIndex(store *client.Client, key [32]byte, cachepath string) (metaIndex, error) {
	listing, err := remote.ListPacks(store)
	if err != nil {
		return nil, err
	}

	err = cleanOldIndexes(listing, cachepath)
	if err != nil {
		return nil, err
	}

	midx := make(metaIndex)
	for _, pack := range listing {
		idx, err := getAndCacheIndex(store, key, pack.Name, pack.Size, cachepath)
		if err != nil {
			return nil, err
		}
		info := &packInfo{
			Name: pack.Name,
			Size: pack.Size,
			Idx:  idx,
		}
		for i := range idx {
			midx[idx[i].Key] = info
		}
	}
	return midx, nil
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
	err = cacheIndex(idxpath, pack.Idx)
	if err != nil {
		return nil, err
	}
	return pack.Idx, nil
}

func cacheIndex(idxpath string, index bpack.Index) error {
	_, err := os.Stat(idxpath)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	tmpname, err := bpy.RandomFileName()
	if err != nil {
		return err
	}
	tmppath := filepath.Join(filepath.Dir(idxpath), tmpname+".tmp")
	tmpf, err := os.OpenFile(tmppath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0400)
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
