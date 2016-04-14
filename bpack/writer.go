package bpack

import (
	"encoding/binary"
	"errors"
	"io"
	"sort"
)

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

func writeSlice(w io.Writer, v []byte) error {
	var lbuf [2]byte

	if len(v) > 65535 {
		return errors.New("value too large for bpack")
	}
	binary.LittleEndian.PutUint16(lbuf[:], uint16(len(v)))

	_, err := w.Write(lbuf[:])
	if err != nil {
		return err
	}
	_, err = w.Write(v)
	return err
}

func (w *Writer) Add(key string, val []byte) error {
	_, has := w.keys[key]
	if has {
		return nil
	}
	err := writeSlice(w.w, val)
	if err != nil {
		return err
	}
	w.keys[key] = struct{}{}
	w.index = append(w.index, IndexEnt{Key: key, Offset: w.offset})
	w.offset += 2 + uint64(len(val))
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
