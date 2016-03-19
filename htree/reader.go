package htree

import (
	"acha.ninja/bpy"
	"errors"
	"io"
)

type Reader struct {
	store  bpy.CStore
	height int
	lvls   [nlevels][maxlen]byte
	pos    [nlevels]int
	length [nlevels]int
}

func NewReader(store bpy.CStore, root [32]byte) (*Reader, error) {
	buf, err := store.Get(root)
	if err != nil {
		return nil, err
	}
	r := &Reader{
		store: store,
	}
	lvl := byte(buf[0])
	r.length[lvl] = len(buf)
	r.pos[lvl] = 1
	for i := range buf {
		r.lvls[0][i] = buf[i]
	}
	return r, nil
}

func (r *Reader) Read(buf []byte) (int, error) {
	nread := 0
	for len(buf) != 0 {
		src := r.lvls[0][r.pos[0]:r.length[0]]
		if len(src) == 0 {
			eof, err := r.next(0)
			if err != nil {
				return nread, err
			}
			if eof {
				return nread, io.EOF
			}
			continue
		}
		n := min(len(buf), len(src))
		for i := 0; i < n; i++ {
			buf[i] = src[i]
		}
		buf = buf[n:]
		r.pos[0] += n
		nread += n
	}
	return nread, nil
}

func (r *Reader) next(lvl int) (bool, error) {
	var hash [32]byte

	if lvl > r.height {
		return false, errors.New("corrupt hash tree: overflowed")
	}
	if r.pos[lvl+1] == r.length[lvl+1] {
		if lvl+1 > r.height {
			return true, nil
		}
		eof, err := r.next(lvl + 1)
		if err != nil {
			return false, err
		}
		if eof {
			return true, nil
		}
	}

	for i := 0; i < len(hash); i++ {
		hash[i] = r.lvls[lvl+1][r.pos[lvl+1]+i]
	}
	buf, err := r.store.Get(hash)
	if err != nil {
		return false, err
	}
	for i := 0; i < len(buf); i++ {
		r.lvls[lvl][i] = buf[i]
	}
	r.length[lvl] = len(buf)
	r.pos[lvl] = 1
	r.pos[lvl+1] += 32
	return false, nil
}
