package cstore

import (
	"acha.ninja/bpy/bpack"
	"container/list"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type lruent struct {
	packname string
	pack     *bpack.Reader
}

type metaIndexEnt struct {
	packname string
	idx      bpack.Index
}

type Reader struct {
	storepath string
	cachepath string
	midx      []metaIndexEnt
	lru       *list.List
}

func NewReader(storepath string, cachepath string) (*Reader, error) {
	dirents, err := ioutil.ReadDir(storepath)
	if err != nil {
		return nil, err
	}
	midx := make([]metaIndexEnt, 0, 16)
	for _, dirent := range dirents {
		if strings.HasSuffix(dirent.Name(), ".bpack") {
			idx, err := getAndCacheIndex(storepath, dirent.Name(), cachepath)
			if err != nil {
				return nil, err
			}
			midxent := metaIndexEnt{
				packname: dirent.Name(),
				idx:      idx,
			}
			midx = append(midx, midxent)
		}
	}
	return &Reader{
		midx:      midx,
		lru:       list.New(),
		storepath: storepath,
		cachepath: cachepath,
	}, nil
}

func (r *Reader) getPackReader(packname string, idx bpack.Index) (*bpack.Reader, error) {
	for e := r.lru.Front(); e != nil; e = e.Next() {
		ent := e.Value.(lruent)
		if ent.packname == packname {
			r.lru.MoveToFront(e)
			return ent.pack, nil
		}
	}
	f, err := os.Open(filepath.Join(r.storepath, packname))
	if err != nil {
		return nil, err
	}
	pack := bpack.NewReader(f)
	pack.Idx = idx
	r.lru.PushFront(lruent{packname: packname, pack: pack})
	if r.lru.Len() > 5 {
		ent := r.lru.Remove(r.lru.Back()).(lruent)
		ent.pack.Close()
	}
	return pack, nil
}

func getAndCacheIndex(storepath, packbasename, cachepath string) (bpack.Index, error) {
	packpath := filepath.Join(storepath, packbasename)
	idxpath := filepath.Join(cachepath, packbasename+".index")

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
	f, err := os.Open(packpath)
	if err != nil {
		return nil, err
	}
	pack := bpack.NewReader(f)
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
