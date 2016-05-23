package cstore

import (
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/proto9"
	"crypto/sha256"
	"encoding/hex"
	"path"
	"snappy"
)

type Writer struct {
	store        *client9.Client
	pack         *bpack.Writer
	cachepath    string
	tmpname      string
	workingSetSz uint64
	midx         []metaIndexEnt
	snappybuf    [65536]byte
}

func NewWriter(store *client9.Client, cachepath string) (*Writer, error) {
	midx, err := readAndCacheMetaIndex(store, cachepath)
	if err != nil {
		return nil, err
	}
	return &Writer{
		cachepath: cachepath,
		midx:      midx,
		store:     store,
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
	dgst := sha256.New()
	for _, ent := range idx {
		_, err := dgst.Write([]byte(ent.Key))
		if err != nil {
			return err
		}
	}
	bpackname := hex.EncodeToString(dgst.Sum(nil)) + ".bpack"
	st := proto9.MaskedStat
	st.Name = bpackname
	err = w.store.Wstat(path.Join("packs", w.tmpname), st)
	if err != nil {
		return err
	}
	err = cacheIndex(bpackname, w.cachepath, idx)
	if err != nil {
		return err
	}
	return nil
}

func (w *Writer) Put(data []byte) ([32]byte, error) {
	var err error

	h := sha256.Sum256(data)
	if w.pack == nil {
		w.tmpname, err = randFileName()
		if err != nil {
			return h, err
		}
		f, err := w.store.Create(path.Join("packs", w.tmpname), 0777, proto9.OWRITE)
		if err != nil {
			return h, err
		}
		w.pack, err = bpack.NewWriter(f)
		if err != nil {
			f.Close()
			w.store.Remove(w.tmpname)
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
	compressed := snappy.Encode(w.snappybuf[:], data)
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
