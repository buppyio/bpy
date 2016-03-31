package htree

import (
	"acha.ninja/bpy"
	"encoding/binary"
)

type Writer struct {
	store  bpy.CStoreWriter
	lvls   [nlevels][maxlen]byte
	nbytes [nlevels]int
	offset uint64
}

func NewWriter(store bpy.CStoreWriter) *Writer {
	w := &Writer{
		store: store,
	}
	w.nbytes[0] = 1
	return w
}

func (w *Writer) Write(buf []byte) (int, error) {
	nbytes := len(buf)
	for len(buf) != 0 {
		n := min(len(buf), maxlen-w.nbytes[0])
		if n == 0 {
			err := w.flushLvl(0)
			if err != nil {
				return 0, err
			}
			w.lvls[0][0] = 0
			w.nbytes[0] = 1
			continue
		}
		copy(w.lvls[0][w.nbytes[0]:maxlen], buf)
		w.nbytes[0] += n
		buf = buf[n:]
	}
	return nbytes, nil
}

func (w *Writer) flushLvl(lvl int) error {
	hash, err := w.store.Put(w.lvls[lvl][0:w.nbytes[lvl]])
	if err != nil {
		return err
	}

	// ensure there is enough room for offset:hash
	if maxlen-w.nbytes[lvl+1] < 8+len(hash) {
		err = w.flushLvl(lvl + 1)
		if err != nil {
			return err
		}
	}

	if w.nbytes[lvl+1] == 0 {
		w.lvls[lvl+1][0] = byte(lvl + 1)
		w.nbytes[lvl+1] = 1
	}

	if lvl == 0 {
		binary.LittleEndian.PutUint64(w.lvls[lvl+1][w.nbytes[lvl+1]:maxlen], w.offset)
	} else {
		copy(w.lvls[lvl+1][w.nbytes[lvl+1]:maxlen], w.lvls[lvl][1:9])
	}
	copy(w.lvls[lvl+1][w.nbytes[lvl+1]+8:maxlen], hash[:])
	w.nbytes[lvl+1] += 8 + len(hash)
	if lvl == 0 {
		w.offset += uint64(w.nbytes[0])
	}
	w.lvls[lvl][0] = byte(lvl)
	w.nbytes[lvl] = 1

	return nil
}

func (w *Writer) Close() ([32]byte, error) {
	highest := 0
	for i := nlevels - 1; ; i-- {
		if w.nbytes[i] != 0 {
			highest = i
			break
		}
	}

	for i := 0; i <= highest; i++ {
		if w.nbytes[i] == 0 {
			continue
		}
		err := w.flushLvl(i)
		if err != nil {
			return [32]byte{}, err
		}
	}

	finalbuf := w.lvls[highest+1][0:w.nbytes[highest+1]]
	return w.store.Put(finalbuf)
}
