package cstore

import (
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/proto9"
	"crypto/sha256"
	"encoding/hex"
	"snappy"
	"sort"
)

type Writer struct {
	rdr          *Reader
	workingSet   map[[32]byte][]byte
	workingSetSz uint64
	store        *client9.Client
	snappybuf    [65536]byte
}

func NewWriter(store *client9.Client, cachepath string) (*Writer, error) {
	rdr, err := NewReader(store, cachepath)
	if err != nil {
		return nil, err
	}
	return &Writer{
		workingSet: make(map[[32]byte][]byte),
		rdr:        rdr,
		store:      store,
	}, nil
}

func (w *Writer) Close() error {
	w.rdr.Close()
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
	bpackname := hex.EncodeToString(dgst.Sum(nil)) + ".bpack"
	_, err := w.store.Stat(bpackname)
	if err == nil {
		return nil
	}
	tmppath := "XXXTODO"
	f, err := w.store.Create(tmppath, 0777, proto9.OWRITE)
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
	st := proto9.MaskedStat
	st.Name = bpackname
	err = w.store.Wstat(tmppath, st)
	if err != nil {
		return err
	}
	midxent := metaIndexEnt{
		packname: bpackname,
		idx:      packidx,
	}
	w.rdr.midx = append(w.rdr.midx, midxent)
	w.workingSet = make(map[[32]byte][]byte)
	w.workingSetSz = 0
	return nil
}

func (w *Writer) Put(data []byte) ([32]byte, error) {
	h := sha256.Sum256(data)
	_, ok := w.workingSet[h]
	if ok {
		return h, nil
	}
	compressed := snappy.Encode(w.snappybuf[:], data)
	_, err := w.rdr.Get(h)
	if err != NotFound {
		return h, err
	}
	v := make([]byte, len(compressed), len(compressed))
	copy(v, compressed)
	w.workingSet[h] = v
	w.workingSetSz += uint64(len(compressed))
	if w.workingSetSz > 1024*1024*128 {
		return h, w.flushWorkingSet()
	} else {
		return h, nil
	}
}
