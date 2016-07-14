package cstore

import (
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/remote/client"
	"bufio"
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"io"
	"io/ioutil"
)

type bufferedWriteCloser struct {
	wc io.WriteCloser
	bw *bufio.Writer
}

func (bwc *bufferedWriteCloser) Write(buf []byte) (int, error) {
	return bwc.bw.Write(buf)
}

func (bwc *bufferedWriteCloser) Close() error {
	err := bwc.bw.Flush()
	if err != nil {
		return err
	}
	return bwc.wc.Close()
}

type Writer struct {
	store        *client.Client
	pack         *bpack.Writer
	cachepath    string
	name         string
	workingSetSz uint64
	key          [32]byte
	midx         []metaIndexEnt
	flatebuf     bytes.Buffer
	flatew       *flate.Writer
}

func NewWriter(store *client.Client, key [32]byte, cachepath string) (*Writer, error) {
	midx, err := readAndCacheMetaIndex(store, key, cachepath)
	if err != nil {
		return nil, err
	}

	flatew, err := flate.NewWriter(ioutil.Discard, flate.BestSpeed)
	if err != nil {
		return nil, err
	}

	return &Writer{
		cachepath: cachepath,
		midx:      midx,
		store:     store,
		key:       key,
		flatew:    flatew,
	}, nil
}

type keyList [][32]byte

func (kl keyList) Len() int           { return len(kl) }
func (kl keyList) Swap(i, j int)      { kl[i], kl[j] = kl[j], kl[i] }
func (kl keyList) Less(i, j int) bool { return bpack.KeyCmp(string(kl[i][:]), string(kl[j][:])) < 0 }

func (w *Writer) flushWorkingSet() error {
	idx, err := w.pack.Close()
	if err != nil {
		return err
	}
	err = cacheIndex(w.name, w.cachepath, idx)
	if err != nil {
		return err
	}
	return nil
}

func (w *Writer) Put(data []byte) ([32]byte, error) {
	var err error

	h := sha256.Sum256(data)
	if w.pack == nil {
		name, err := randFileName()
		if err != nil {
			return h, err
		}
		f, err := w.store.NewPack("packs/" + name + ".ebpack")
		if err != nil {
			return h, err
		}
		bwc := &bufferedWriteCloser{
			wc: f,
			bw: bufio.NewWriterSize(f, 65536),
		}
		w.pack, err = bpack.NewEncryptedWriter(bwc, w.key)
		if err != nil {
			f.Cancel()
			return h, err
		}
	}
	if w.pack.Has(string(h[:])) {
		return h, nil
	}
	_, _, ok := searchMetaIndex(w.midx, h)
	if ok {
		return h, nil
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
	if w.workingSetSz > 1024*1024*128 {
		return h, w.flushWorkingSet()
	} else {
		return h, nil
	}
}

func (w *Writer) Close() error {
	return w.flushWorkingSet()
}
