package cstore

import (
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/proto9"
	"bytes"
	"compress/flate"
	"container/list"
	"errors"
	"io"
	"path"
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
	key       [32]byte
	flatebuf  bytes.Buffer
}

func NewReader(store *client9.Client, key [32]byte, cachepath string) (*Reader, error) {
	midx, err := readAndCacheMetaIndex(store, key, cachepath)
	if err != nil {
		return nil, err
	}
	return &Reader{
		midx:      midx,
		lru:       list.New(),
		store:     store,
		cachepath: cachepath,
		key:       key,
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
	bufrdr := bytes.NewReader(buf)
	compressedr := flate.NewReader(bufrdr)
	if err != nil {
		return nil, err
	}
	r.flatebuf.Reset()
	_, err = io.Copy(&r.flatebuf, compressedr)
	if err != nil {
		return nil, err
	}
	err = compressedr.Close()
	if err != nil {
		return nil, err
	}
	decompressed := make([]byte, r.flatebuf.Len(), r.flatebuf.Len())
	copy(decompressed, r.flatebuf.Bytes())
	return decompressed, nil
}

func (r *Reader) getPackReader(packname string, idx bpack.Index) (*bpack.Reader, error) {
	for e := r.lru.Front(); e != nil; e = e.Next() {
		ent := e.Value.(lruent)
		if ent.packname == packname {
			r.lru.MoveToFront(e)
			return ent.pack, nil
		}
	}
	packPath := path.Join("packs", packname)
	stat, err := r.store.Stat(packPath)
	if err != nil {
		return nil, err
	}
	f, err := r.store.Open(packPath, proto9.OREAD)
	if err != nil {
		return nil, err
	}
	pack, err := bpack.NewEncryptedReader(f, r.key, int64(stat.Length))
	if err != nil {
		return nil, err
	}
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
