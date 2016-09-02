package refs

import (
	"errors"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/htree"
	"io/ioutil"
)

var (
	ErrInvalidRef = errors.New("invalid ref")
)

type Ref struct {
	Root    [32]byte
	HasPrev bool
	Prev    [32]byte
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
	switch len(data) {
	case 32:
		ref := Ref{}
		copy(ref.Root[:], data)
		return ref, nil
	case 64:
		ref := Ref{}
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
	_, err := w.Write(ref.Root[:])
	if err != nil {
		w.Close()
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
