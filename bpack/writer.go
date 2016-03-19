package bpack

import (
	"encoding/binary"
	"errors"
	"io"
	"sort"
)

type Writer struct {
	w      io.WriteSeeker
	keys   map[string]struct{}
	index  Index
	offset uint64
}

func NewWriter(w io.WriteSeeker) (*Writer, error) {
	var zero [8]byte

	_, err := w.Write(zero[:])
	return &Writer{
		w:      w,
		keys:   make(map[string]struct{}),
		index:  make(Index, 0, 2048),
		offset: 8,
	}, err
}

func (w *Writer) writeUInt64(v uint64) error {
	var buf [8]byte

	binary.LittleEndian.PutUint64(buf[:], v)
	_, err := w.w.Write(buf[:])
	return err
}

func (w *Writer) writeSlice(v []byte) error {
	var lbuf [2]byte

	if len(v) > 65535 {
		return errors.New("value too large for bpack")
	}
	binary.LittleEndian.PutUint16(lbuf[:], uint16(len(v)))

	_, err := w.w.Write(lbuf[:])
	if err != nil {
		return err
	}
	_, err = w.w.Write(v)
	return err
}

func (w *Writer) Add(key string, val []byte) error {
	_, has := w.keys[key]
	if has {
		return nil
	}
	err := w.writeSlice(val)
	if err != nil {
		return err
	}
	w.keys[key] = struct{}{}
	w.index = append(w.index, IndexEnt{Key: key, Offset: w.offset})
	w.offset += 2 + uint64(len(val))
	return nil
}

func (w *Writer) Close() error {
	sort.Sort(w.index)
	idxoffset := w.offset
	err := w.writeUInt64(uint64(len(w.index)))
	if err != nil {
		return err
	}
	for i := range w.index {
		err = w.writeSlice([]byte(w.index[i].Key))
		if err != nil {
			return err
		}
		err = w.writeUInt64(w.index[i].Offset)
		if err != nil {
			return err
		}
	}
	_, err = w.w.Seek(0, 0)
	if err != nil {
		return err
	}
	err = w.writeUInt64(idxoffset)
	if err != nil {
		return err
	}
	return nil
}
