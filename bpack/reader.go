package bpack

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
)

var NotFound = errors.New("Not Found")

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type Reader struct {
	r    ReadSeekCloser
	size uint64
	Idx  Index
}

func NewReader(r ReadSeekCloser, size uint64) *Reader {
	return &Reader{
		r:    r,
		size: size,
	}
}

func (r *Reader) Get(key string) ([]byte, error) {
	idx, ok := r.Idx.Search(key)
	if !ok {
		return nil, NotFound
	}
	off := r.Idx[idx].Offset
	sz := r.Idx[idx].Size
	return r.GetAt(off, sz)
}

func (r *Reader) GetAt(offset uint64, sz uint32) ([]byte, error) {
	buf := make([]byte, sz, sz)
	r.r.Seek(int64(offset), io.SeekStart)
	_, err := io.ReadFull(r.r, buf)
	return buf, err
}

func (r *Reader) Close() error {
	return r.r.Close()
}

func (r *Reader) ReadIndex() error {
	_, err := r.r.Seek(int64(r.size)-8, io.SeekStart)
	if err != nil {
		return err
	}
	offset, err := readUint64(r.r)
	if err != nil {
		return err
	}
	_, err = r.r.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return err
	}
	r.Idx, err = ReadIndex(r.r)
	return err
}

func readSlice(r io.Reader) ([]byte, error) {
	l, err := readUint24(r)
	if err != nil {
		return nil, err
	}
	ret := make([]byte, l, l)
	_, err = io.ReadFull(r, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func readUint64(r io.Reader) (uint64, error) {
	var buf [8]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return 0, err
	}
	n := binary.LittleEndian.Uint64(buf[:])
	return n, nil
}

func readUint24(r io.Reader) (uint32, error) {
	var buf [4]byte
	_, err := io.ReadFull(r, buf[0:3])
	if err != nil {
		return 0, err
	}
	n := binary.LittleEndian.Uint32(buf[:])
	return n, nil
}

func ReadIndex(r io.Reader) (Index, error) {
	r = bufio.NewReaderSize(r, 65536)
	idx := make(Index, 0, 4096)
	n, err := readUint64(r)
	if err != nil {
		return idx, err
	}
	for n != 0 {
		ksl, err := readSlice(r)
		if err != nil {
			return idx, err
		}
		sz, err := readUint24(r)
		if err != nil {
			return idx, err
		}
		offset, err := readUint64(r)
		if err != nil {
			return idx, err
		}
		idx = append(idx, IndexEnt{Key: string(ksl), Offset: offset, Size: sz})
		n--
	}
	return idx, nil
}
