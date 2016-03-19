package fs

import (
	"testing"
	"reflect"
	"crypto/sha256"
	"errors"
)

type memstore struct {
	vals map[string][]byte
}

func (m *memstore) Get(hash [32]byte) ([]byte, error) {
	val, ok := m.vals[string(hash[:])]
	if !ok {
		return nil, errors.New("hash not found in store")
	}
	return val, nil
}

func (m *memstore) Put(val []byte) ([32]byte, error) {
	hash := sha256.Sum256(val)
	valcpy := make([]byte, len(val), len(val))
	copy(valcpy, val)
	m.vals[string(hash[:])] = valcpy
	return hash, nil
}

func TestDir(t *testing.T) {
	dir := DirEnts{
		{Name: "Bar", Size: 4, Mode: 5, ModTime: 6, Data: [32]byte{1,2,3,4}},
		{Name: "Foo", Size: 0xffffff, Mode: 0xffffff, ModTime: 0xffff, },
	}
	store := &memstore{vals: make(map[string][]byte)}
	
	hash, err := WriteDir(store, dir)
	if err != nil {
		t.Fatal(err)
	}
	rdir, err := ReadDir(store, hash)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dir, rdir) {
		t.Fatalf("dirs differ\n%v\n%v\n", dir, rdir)
	}
}
