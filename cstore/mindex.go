package cstore

import (
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/proto9"
	"os"
	"path/filepath"
	"strings"
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

func readAndCacheMetaIndex(store *client9.Client, cachepath string) ([]metaIndexEnt, error) {
	dirents, err := store.Ls("/")
	if err != nil {
		return nil, err
	}
	midx := make([]metaIndexEnt, 0, 16)
	for _, dirent := range dirents {
		if strings.HasSuffix(dirent.Name, ".bpack") {
			idx, err := getAndCacheIndex(store, dirent.Name, cachepath)
			if err != nil {
				return nil, err
			}
			midxent := metaIndexEnt{
				packname: dirent.Name,
				idx:      idx,
			}
			midx = append(midx, midxent)
		}
	}
	return midx, nil
}

func getAndCacheIndex(store *client9.Client, packname, cachepath string) (bpack.Index, error) {
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
	stat, err := store.Stat(packname)
	if err != nil {
		return nil, err
	}
	f, err := store.Open(packname, proto9.OREAD)
	if err != nil {
		return nil, err
	}
	pack := bpack.NewReader(f, stat.Length)
	defer pack.Close()
	err = pack.ReadIndex()
	if err != nil {
		return nil, err
	}
	idxf, err := os.Create(idxpath)
	if err != nil {
		return nil, err
	}
	err = bpack.WriteIndex(idxf, pack.Idx)
	if err != nil {
		return nil, err
	}
	return pack.Idx, idxf.Close()
}
