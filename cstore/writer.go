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
	workingSet   map[string][]byte
	workingSetSz uint64
	storepath    string
}

func NewWriter(storepath string, cachepath string) (*Writer, error) {
	rdr, err := NewReader(storepath, cachepath)
	if err != nil {
		return nil, err
	}
	return &Writer{
		workingSet: make(map[string][]byte),
		rdr:        rdr,
		storepath:  storepath,
	}, nil
}

func (w *Writer) Close() error {
	return w.flushWorkingSet()
}

type keyList []string

func (kl keyList) Len() int           { return len(kl) }
func (kl keyList) Swap(i, j int)      { kl[i], kl[j] = kl[j], kl[i] }
func (kl keyList) Less(i, j int) bool { return bpack.KeyCmp(kl[i], kl[j]) < 0 }

func (w *Writer) flushWorkingSet() error {
	if len(w.workingSet) == 0 {
		return nil
	}
	keys := make(keyList, len(w.workingSet), len(w.workingSet))
	i := 0
	for k := range w.workingSet {
		keys[i] = k
	}
	sort.Sort(keys)
	dgst := sha256.New()
	for _, k := range keys {
		_, err := dgst.Write([]byte(k))
		if err != nil {
			return err
		}
	}
	bpackname := filepath.Join(w.storepath, hex.EncodeToString(dgst.Sum(nil))+".bpack")
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
		err = pack.Add(k, w.workingSet[k])
		if err != nil {
			return err
		}
	}
	err = pack.Close()
	if err != nil {
		return err
	}
	return os.Rename(tmppath, bpackname)
}

func (w *Writer) Add(data []byte) ([32]byte, error) {
	h := sha256.Sum256(data)
	k := string(h[:])
	_, ok := w.workingSet[k]
	if ok {
		return h, nil
	}
	v := make([]byte, len(data), len(data))
	copy(v, data)
	w.workingSet[k] = v
	w.workingSetSz += uint64(len(data))
	if w.workingSetSz > 1024*1024*128 {
		return h, w.flushWorkingSet()
	} else {
		return h, nil
	}
}
