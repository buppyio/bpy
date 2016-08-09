package cstore

import (
	"bytes"
	"compress/flate"
	"container/list"
	"errors"
	"github.com/buppyio/bpy/bpack"
	"github.com/buppyio/bpy/remote/client"
	"io"
	"path"
	"sync"
)

var NotFound = errors.New("hash not in cstore")

type packlruent struct {
	packname string
	pack     *bpack.Reader
}

type Reader struct {
	lock      sync.Mutex
	store     *client.Client
	cachepath string
	midx      metaIndex
	lru       *list.List
	key       [32]byte
	flatebuf  bytes.Buffer
}

func NewReader(store *client.Client, key [32]byte, cachepath string) (*Reader, error) {
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

func (r *Reader) Has(hash [32]byte) (bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	_, _, ok := searchMetaIndex(r.midx, hash)
	return ok, nil
}

func (r *Reader) Get(hash [32]byte) ([]byte, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	packInfo, packidxent, ok := searchMetaIndex(r.midx, hash)
	if !ok {
		midx, err := readAndCacheMetaIndex(r.store, r.key, r.cachepath)
		if err != nil {
			return nil, err
		}
		r.midx = midx
		packInfo, packidxent, ok = searchMetaIndex(r.midx, hash)
		if !ok {
			return nil, NotFound
		}
	}

	packrdr, err := r.getPackReader(packInfo.Name, packInfo.Size, packInfo.Idx)
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

func (r *Reader) getPackReader(packname string, packsize uint64, idx bpack.Index) (*bpack.Reader, error) {
	for e := r.lru.Front(); e != nil; e = e.Next() {
		ent := e.Value.(packlruent)
		if ent.packname == packname {
			r.lru.MoveToFront(e)
			return ent.pack, nil
		}
	}
	packPath := path.Join("packs", packname)
	f, err := r.store.Open(packPath)
	if err != nil {
		return nil, err
	}
	pack, err := bpack.NewEncryptedReader(f, r.key, int64(packsize))
	if err != nil {
		return nil, err
	}
	pack.Idx = idx
	r.lru.PushFront(packlruent{packname: packname, pack: pack})
	if r.lru.Len() > 5 {
		ent := r.lru.Remove(r.lru.Back()).(packlruent)
		ent.pack.Close()
	}
	return pack, nil
}

func (r *Reader) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	for e := r.lru.Front(); e != nil; e = e.Next() {
		ent := e.Value.(packlruent)
		err := ent.pack.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
