package bpack

import (
	"encoding/binary"
	"errors"
	"io"
	"sort"
)

const MaxPackEntrySize = 16777215

type Writer struct {
	w      io.WriteCloser
	keys   map[string]struct{}
	index  Index
	offset uint64
}

func NewWriter(w io.WriteCloser) (*Writer, error) {
	return &Writer{
		w:      w,
		keys:   make(map[string]struct{}),
		index:  make(Index, 0, 2048),
		offset: 0,
	}, nil
}

func writeUInt64(w io.Writer, v uint64) error {
	var buf [8]byte

	binary.LittleEndian.PutUint64(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

func writeUInt24(w io.Writer, v uint32) error {
	var buf [4]byte

	binary.LittleEndian.PutUint32(buf[:], v)
	_, err := w.Write(buf[0:3])
	return err
}

func writeSlice(w io.Writer, v []byte) error {
	if len(v) > MaxPackEntrySize {
		return errors.New("value too large for bpack")
	}
	err := writeUInt24(w, uint32(len(v)))
	if err != nil {
		return err
	}
	_, err = w.Write(v)
	return err
}

func (w *Writer) Has(key string) bool {
	_, has := w.keys[key]
	return has
}

func (w *Writer) Add(key string, val []byte) error {
	_, has := w.keys[key]
	if has {
		return nil
	}
	if len(val) > MaxPackEntrySize {
		return errors.New("value too large for bpack")
	}
	_, err := w.w.Write(val)
	if err != nil {
		return err
	}
	w.keys[key] = struct{}{}
	w.index = append(w.index, IndexEnt{Key: key, Offset: w.offset, Size: uint32(len(val))})
	w.offset += uint64(len(val))
	return nil
}

func WriteIndex(w io.Writer, idx Index) error {
	sort.Sort(idx)
	err := writeUInt64(w, uint64(len(idx)))
	if err != nil {
		return err
	}
	for i := range idx {
		err = writeSlice(w, []byte(idx[i].Key))
		if err != nil {
			return err
		}
		err = writeUInt24(w, idx[i].Size)
		if err != nil {
			return err
		}
		err = writeUInt64(w, idx[i].Offset)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) Close() (Index, error) {
	idxoffset := w.offset
	err := WriteIndex(w.w, w.index)
	if err != nil {
		return nil, err
	}
	err = writeUInt64(w.w, idxoffset)
	if err != nil {
		return nil, err
	}
	return w.index, w.w.Close()
}
