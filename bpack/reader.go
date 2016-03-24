package bpack

import (
	"encoding/binary"
	"io"
)

type Reader struct {
	r     io.ReadSeeker
	Idx Index
}

func NewReader(r io.ReadSeeker) *Reader {
	return &Reader{
		r: r,
	}
}

func (r *Reader) Get(key string) ([]byte, bool, error) {
	lo := 0
	hi := len(r.Idx) - 1
	idx := -1
	for lo <= hi {
		mid := (hi + lo) / 2
		switch keycmp(r.Idx[mid].Key, key) {
		case 1:
			hi = mid - 1
		case -1:
			lo = mid + 1
		case 0:
			idx = mid
			goto done
		}
	}
done:
	if idx == -1 {
		return nil, false, nil
	}
	off := r.Idx[idx].Offset
	r.r.Seek(int64(off), 0)
	b, err := readSlice(r.r)
	return b, true, err
}

func (r *Reader) ReadIndex() error {
	_, err := r.r.Seek(0, 0)
	if err != nil {
		return err
	}
	offset, err := readUint64(r.r)
	if err != nil {
		return err
	}
	_, err = r.r.Seek(int64(offset), 0)
	if err != nil {
		return err
	}
	r.Idx, err = ReadIndex(r.r)
	return err
}

func readSlice(r io.Reader) ([]byte, error) {
	var ret []byte
	var buf [2]byte
	_, err := r.Read(buf[:])
	if err != nil {
		return ret, err
	}
	l := binary.LittleEndian.Uint16(buf[:2])
	ret = make([]byte, l, l)
	_, err = r.Read(ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func readUint64(r io.Reader) (uint64, error) {
	var buf [8]byte
	_, err := r.Read(buf[:])
	if err != nil {
		return 0, err
	}
	n := binary.LittleEndian.Uint64(buf[:])
	return n, nil
}

func ReadIndex(r io.Reader) (Index, error) {
	idx := make(Index, 0, 2048)
	n, err := readUint64(r)
	if err != nil {
		return idx, err
	}
	for n != 0 {
		ksl, err := readSlice(r)
		if err != nil {
			return idx, err
		}
		offset, err := readUint64(r)
		if err != nil {
			return idx, err
		}
		idx = append(idx, IndexEnt{Key: string(ksl), Offset: offset})
		n--
	}
	return idx, nil
}
