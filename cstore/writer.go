package cstore

import (
	"bufio"
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/bpack"
	"github.com/buppyio/bpy/remote/client"
	"io/ioutil"
	"path/filepath"
	"sync"
)

type Writer struct {
	lock         sync.Mutex
	store        *client.Client
	pack         *bpack.Writer
	cachepath    string
	name         string
	workingSet   map[string][]byte
	workingSetSz uint64
	key          [32]byte
	flatebuf     bytes.Buffer
	flatew       *flate.Writer
	rdr          *Reader
}

func NewWriter(store *client.Client, key [32]byte, cachepath string) (*Writer, error) {
	rdr, err := NewReader(store, key, cachepath)
	if err != nil {
		return nil, err
	}

	flatew, err := flate.NewWriter(ioutil.Discard, flate.BestSpeed)
	if err != nil {
		return nil, err
	}

	return &Writer{
		cachepath:  cachepath,
		store:      store,
		key:        key,
		flatew:     flatew,
		rdr:        rdr,
		workingSet: make(map[string][]byte),
	}, nil
}

type keyList [][32]byte

func (kl keyList) Len() int           { return len(kl) }
func (kl keyList) Swap(i, j int)      { kl[i], kl[j] = kl[j], kl[i] }
func (kl keyList) Less(i, j int) bool { return bpack.KeyCmp(string(kl[i][:]), string(kl[j][:])) < 0 }

func (w *Writer) flushWorkingSet() error {
	if w.pack != nil {
		idx, err := w.pack.Close()
		if err != nil {
			return err
		}
		err = cacheIndex(filepath.Join(w.cachepath, w.name+".index"), idx)
		if err != nil {
			return err
		}
		w.pack = nil
		w.name = ""
		w.workingSetSz = 0
		w.workingSet = make(map[string][]byte)
	}
	return nil
}

func (w *Writer) Get(hash [32]byte) ([]byte, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	val, ok := w.workingSet[string(hash[:])]
	if ok {
		return val, nil
	}
	return w.rdr.Get(hash)
}

func (w *Writer) Has(hash [32]byte) (bool, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	_, ok := w.workingSet[string(hash[:])]
	if ok {
		return true, nil
	}
	return w.rdr.Has(hash)
}

func (w *Writer) Put(data []byte) ([32]byte, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	var err error

	h := sha256.Sum256(data)

	_, ok := w.workingSet[string(h[:])]
	if ok {
		return h, nil
	}

	ok, err = w.rdr.Has(h)
	if err != nil {
		return h, err
	}
	if ok {
		return h, nil
	}

	if w.pack == nil {
		name, err := bpy.RandomFileName()
		if err != nil {
			return h, err
		}
		name = name + ".ebpack"
		f, err := w.store.NewPack("packs/" + name)
		if err != nil {
			return h, err
		}
		bwc := &bpy.BufferedWriteCloser{
			W: f,
			B: bufio.NewWriterSize(f, 65536),
		}
		w.pack, err = bpack.NewEncryptedWriter(bwc, w.key)
		if err != nil {
			f.Cancel()
			return h, err
		}
		w.name = name
	}

	w.flatebuf.Reset()
	w.flatew.Reset(&w.flatebuf)
	_, err = w.flatew.Write(data)
	if err != nil {
		return h, err
	}
	err = w.flatew.Close()
	if err != nil {
		return h, err
	}
	compressed := w.flatebuf.Bytes()
	err = w.pack.Add(string(h[:]), compressed)
	if err != nil {
		return h, err
	}
	w.workingSetSz += uint64(len(compressed))
	dataCopy := make([]byte, len(data), len(data))
	copy(dataCopy, data)
	w.workingSet[string(h[:])] = dataCopy
	if w.workingSetSz > 1024*1024*128 {
		return h, w.flushWorkingSet()
	} else {
		return h, nil
	}
}

func (w *Writer) Flush() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.flushWorkingSet()
}

func (w *Writer) Close() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.flushWorkingSet()
}
