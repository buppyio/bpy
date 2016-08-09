package htree

import (
	"encoding/binary"
	"errors"
	"github.com/buppyio/bpy"
	"io"
)

type Reader struct {
	root   [32]byte
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
		root:  root,
		store: store,
	}
	lvl := int(buf[0])
	r.length[lvl] = len(buf)
	r.pos[lvl] = 1
	r.height = lvl
	copy(r.lvls[lvl][:], buf)
	return r, nil
}

func (r *Reader) Seek(absoff uint64) (uint64, error) {
	buf, err := r.store.Get(r.root)
	if err != nil {
		return 0, err
	}
	lvl := int(buf[0])
	r.length[lvl] = len(buf)
	r.pos[lvl] = 1
	r.height = lvl
	copy(r.lvls[lvl][:], buf)
	curoff := uint64(0)
	for lvl != 0 {
		for {
			var enthash [32]byte
			curoff = binary.LittleEndian.Uint64(r.lvls[lvl][r.pos[lvl]:])
			copy(enthash[:], r.lvls[lvl][r.pos[lvl]+8:])
			r.pos[lvl] += 40
			nextoff := absoff + 1
			if r.pos[lvl] < r.length[lvl] {
				nextoff = binary.LittleEndian.Uint64(r.lvls[lvl][r.pos[lvl]:])
			}
			if absoff < nextoff {
				buf, err := r.store.Get(enthash)
				if err != nil {
					// XXX revert seek on error?
					return 0, err
				}
				lvl -= 1
				r.length[lvl] = len(buf)
				r.pos[lvl] = 1
				copy(r.lvls[lvl][:], buf)
				break
			}
		}
	}
	skipamnt := int(absoff - curoff)
	if skipamnt+r.pos[0] > r.length[0] {
		absoff -= uint64((skipamnt + r.pos[0]) - r.length[0])
		r.pos[0] = r.length[0]
	} else {
		r.pos[0] += int(skipamnt)
	}
	return absoff, nil

}

func (r *Reader) Read(buf []byte) (int, error) {
	src := r.lvls[0][r.pos[0]:r.length[0]]
	if len(src) == 0 {
		eof, err := r.next(0)
		if err != nil {
			return 0, err
		}
		if eof {
			return 0, io.EOF
		}
		src = r.lvls[0][r.pos[0]:r.length[0]]
	}
	n := copy(buf, src)
	r.pos[0] += n
	return n, nil
}

func (r *Reader) next(lvl int) (bool, error) {
	var hash [32]byte

	if lvl > r.height {
		return false, errors.New("corrupt hash tree: overflowed")
	}
	if r.pos[lvl+1] >= r.length[lvl+1] {
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
	copy(hash[:], r.lvls[lvl+1][r.pos[lvl+1]+8:maxlen])
	buf, err := r.store.Get(hash)
	if err != nil {
		return false, err
	}
	copy(r.lvls[lvl][0:len(buf)], buf)
	r.pos[lvl+1] += 40
	r.length[lvl] = len(buf)
	r.pos[lvl] = 1
	return false, nil
}
