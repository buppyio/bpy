package htree

import (
	"acha.ninja/bpy"
)

type Writer struct {
	store  bpy.CStore
	lvls   [nlevels][maxlen]byte
	nbytes [nlevels]int
}

func NewWriter(store bpy.CStore) *Writer {
	return &Writer{
		store: store,
	}
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
		for i := 0; i < n; i++ {
			w.lvls[0][w.nbytes[0]+i] = buf[i]
		}
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

	// ensure there is enough room for the hash
	if len(w.lvls[lvl+1])-w.nbytes[lvl+1] < len(hash) {
		err = w.flushLvl(lvl + 1)
		if err != nil {
			return err
		}
	}

	if w.nbytes[lvl+1] == 0 {
		w.lvls[lvl+1][0] = byte(lvl + 1)
		w.nbytes[lvl+1] = 1
	}

	for i := range hash {
		w.lvls[lvl+1][w.nbytes[lvl+1]+i] = hash[i]
	}

	w.nbytes[lvl+1] += len(hash)
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
	}

	finalbuf := w.lvls[highest+1][0:w.nbytes[highest+1]]
	return w.store.Put(finalbuf)
}
