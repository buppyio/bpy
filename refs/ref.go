package refs

import (
	"encoding/binary"
	"errors"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/htree"
	"io/ioutil"
	"time"
)

var (
	ErrInvalidRef = errors.New("invalid ref")
)

type Ref struct {
	CreatedAt int64
	Root      [32]byte
	HasPrev   bool
	Prev      [32]byte
}

func GetRef(store bpy.CStore, hash [32]byte) (Ref, error) {
	rdr, err := htree.NewReader(store, hash)
	if err != nil {
		return Ref{}, nil
	}
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		return Ref{}, nil
	}

	createdAt := int64(binary.LittleEndian.Uint64(data[0:8]))
	data = data[8:]

	switch len(data) {
	case 32:
		ref := Ref{}
		ref.CreatedAt = createdAt
		copy(ref.Root[:], data)
		return ref, nil
	case 64:
		ref := Ref{}
		ref.CreatedAt = createdAt
		copy(ref.Root[:], data[0:32])
		copy(ref.Prev[:], data[32:64])
		ref.HasPrev = true
		return ref, nil
	default:
		return Ref{}, ErrInvalidRef
	}
}

func PutRef(store bpy.CStore, ref Ref) ([32]byte, error) {
	w := htree.NewWriter(store)
	defer w.Close()

	var t [8]byte
	binary.LittleEndian.PutUint64(t[:], uint64(ref.CreatedAt))
	_, err := w.Write(t[:])
	if err != nil {
		return [32]byte{}, err
	}

	_, err = w.Write(ref.Root[:])
	if err != nil {
		return [32]byte{}, err
	}

	if ref.HasPrev {
		_, err := w.Write(ref.Prev[:])
		if err != nil {
			w.Close()
			return [32]byte{}, err
		}
	}
	tree, err := w.Close()
	return tree.Data, err
}

func GetAtTime(store bpy.CStore, ref Ref, at time.Time) (Ref, bool, error) {
	atUnix := at.Unix()
	for {
		if atUnix >= ref.CreatedAt {
			return ref, true, nil
		}

		if ref.HasPrev == false {
			return Ref{}, false, nil
		}

		prevRef, err := GetRef(store, ref.Prev)
		if err != nil {
			return Ref{}, false, err
		}
		ref = prevRef
	}
}

func GetNVersionsAgo(store bpy.CStore, ref Ref, n uint64) (Ref, error) {
	for n != 0 {
		if ref.HasPrev == false {
			break
		}
		nextRef, err := GetRef(store, ref.Prev)
		if err != nil {
			return Ref{}, err
		}
		ref = nextRef
		n -= 1
	}
	return ref, nil
}
