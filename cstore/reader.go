package cstore

import (
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/proto9"
	"container/list"
	"errors"
	"snappy"
)

var NotFound = errors.New("hash not in cstore")

type lruent struct {
	packname string
	pack     *bpack.Reader
}

type metaIndexEnt struct {
	packname string
	idx      bpack.Index
}

type Reader struct {
	store     *client9.Client
	cachepath string
	midx      []metaIndexEnt
	lru       *list.List
}

func NewReader(store *client9.Client, cachepath string) (*Reader, error) {
	midx, err := readAndCacheMetaIndex(store, cachepath)
	if err != nil {
		return nil, err
	}
	return &Reader{
		midx:      midx,
		lru:       list.New(),
		store:     store,
		cachepath: cachepath,
	}, nil
}

func (r *Reader) Get(hash [32]byte) ([]byte, error) {
	midxent, packidxent, ok := searchMetaIndex(r.midx, hash)
	if !ok {
		return nil, NotFound
	}
	packrdr, err := r.getPackReader(midxent.packname, midxent.idx)
	if err != nil {
		return nil, err
	}
	buf, err := packrdr.GetAt(packidxent.Offset, packidxent.Size)
	if err != nil {
		return nil, err
	}
	return snappy.Decode(nil, buf)
}

func (r *Reader) getPackReader(packname string, idx bpack.Index) (*bpack.Reader, error) {
	for e := r.lru.Front(); e != nil; e = e.Next() {
		ent := e.Value.(lruent)
		if ent.packname == packname {
			r.lru.MoveToFront(e)
			return ent.pack, nil
		}
	}
	stat, err := r.store.Stat(packname)
	if err != nil {
		return nil, err
	}
	f, err := r.store.Open(packname, proto9.OREAD)
	if err != nil {
		return nil, err
	}
	pack := bpack.NewReader(f, stat.Length)
	pack.Idx = idx
	r.lru.PushFront(lruent{packname: packname, pack: pack})
	if r.lru.Len() > 5 {
		ent := r.lru.Remove(r.lru.Back()).(lruent)
		ent.pack.Close()
	}
	return pack, nil
}

func (r *Reader) Close() error {
	for e := r.lru.Front(); e != nil; e = e.Next() {
		ent := e.Value.(lruent)
		err := ent.pack.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
