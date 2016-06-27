package cstore

import (
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/proto9"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path"
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

func readAndCacheMetaIndex(store *client9.Client, key [32]byte, cachepath string) ([]metaIndexEnt, error) {
	dirents, err := store.Ls("packs")
	if err != nil {
		return nil, err
	}
	midx := make([]metaIndexEnt, 0, 16)
	for _, dirent := range dirents {
		if strings.HasSuffix(dirent.Name, ".bpack") {
			idx, err := getAndCacheIndex(store, key, dirent.Name, cachepath)
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

func getAndCacheIndex(store *client9.Client, key [32]byte, packname, cachepath string) (bpack.Index, error) {
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
	stat, err := store.Stat(path.Join("packs", packname))
	if err != nil {
		return nil, err
	}
	f, err := store.Open(path.Join("packs", packname), proto9.OREAD)
	if err != nil {
		return nil, err
	}
	pack, err := bpack.NewEncryptedReader(f, key, int64(stat.Length))
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
