package htree

import (
	"acha.ninja/bpy"
	"encoding/binary"
	"errors"
	"io"
	"fmt"
)

type Reader struct {
	root   [32]byte
	store  bpy.CStoreReader
	height int
	lvls   [nlevels][maxlen]byte
	pos    [nlevels]int
	length [nlevels]int
	offset uint64
}

func NewReader(store bpy.CStoreReader, root [32]byte) (*Reader, error) {
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

func (r *Reader) Seek(offset int64) (int64, error) {
	// XXX todo proper seek
	absoff := uint64(offset)
	fmt.Printf("seeking to %d\n", absoff)
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
		fmt.Printf("seeking lvl %d\n", lvl)
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
	r.pos[0] += int(absoff - curoff)
	return int64(absoff), nil

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
		copy(buf, src)
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
	r.pos[lvl+1] += 32 + 8
	r.length[lvl] = len(buf)
	r.pos[lvl] = 1
	return false, nil
}
