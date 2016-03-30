package cstore

import (
	"acha.ninja/bpy/bpack"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
)

type Writer struct {
	rdr          *Reader
	workingSet   map[[32]byte][]byte
	workingSetSz uint64
	storepath    string
}

func NewWriter(storepath string, cachepath string) (*Writer, error) {
	rdr, err := NewReader(storepath, cachepath)
	if err != nil {
		return nil, err
	}
	return &Writer{
		workingSet: make(map[[32]byte][]byte),
		rdr:        rdr,
		storepath:  storepath,
	}, nil
}

func (w *Writer) Close() error {
	return w.flushWorkingSet()
}

type keyList [][32]byte

func (kl keyList) Len() int           { return len(kl) }
func (kl keyList) Swap(i, j int)      { kl[i], kl[j] = kl[j], kl[i] }
func (kl keyList) Less(i, j int) bool { return bpack.KeyCmp(string(kl[i][:]), string(kl[j][:])) < 0 }

func (w *Writer) flushWorkingSet() error {
	if len(w.workingSet) == 0 {
		return nil
	}
	keys := make(keyList, len(w.workingSet), len(w.workingSet))
	i := 0
	for k := range w.workingSet {
		keys[i] = k
		i++
	}
	sort.Sort(keys)
	dgst := sha256.New()
	for _, k := range keys {
		_, err := dgst.Write(k[:])
		if err != nil {
			return err
		}
	}
	bpackbasename := hex.EncodeToString(dgst.Sum(nil)) + ".bpack"
	bpackname := filepath.Join(w.storepath, bpackbasename)
	_, err := os.Stat(bpackname)
	if err == nil {
		return nil
	}
	tmppath := filepath.Join(w.storepath, "XXXTODO.bpack")
	f, err := os.Create(tmppath)
	if err != nil {
		return err
	}
	pack, err := bpack.NewWriter(f)
	if err != nil {
		return err
	}
	for _, k := range keys {
		err = pack.Add(string(k[:]), w.workingSet[k])
		if err != nil {
			return err
		}
	}
	packidx, err := pack.Close()
	if err != nil {
		return err
	}
	err = os.Rename(tmppath, bpackname)
	if err != nil {
		return err
	}
	midxent := metaIndexEnt{
		packname: bpackbasename,
		idx:      packidx,
	}
	w.rdr.midx = append(w.rdr.midx, midxent)
	return nil
}

func (w *Writer) Put(data []byte) ([32]byte, error) {
	h := sha256.Sum256(data)
	_, ok := w.workingSet[h]
	if ok {
		return h, nil
	}
	v := make([]byte, len(data), len(data))
	copy(v, data)
	w.workingSet[h] = v
	w.workingSetSz += uint64(len(data))
	if w.workingSetSz > 1024*1024*128 {
		return h, w.flushWorkingSet()
	} else {
		return h, nil
	}
}
