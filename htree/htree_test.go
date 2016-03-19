package htree

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"math/rand"
	"testing"
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

func TestHTree(t *testing.T) {
	for i := 0; i < 50; i++ {
		var randbytes bytes.Buffer
		var readbytes bytes.Buffer

		store := &memstore{vals: make(map[string][]byte)}
		rand := rand.New(rand.NewSource(int64(i + 100)))
		random := &io.LimitedReader{N: int64(rand.Int31() % 5 * 1024 * 1024), R: rand}
		_, err := io.Copy(&randbytes, random)
		if err != nil {
			t.Fatal(err)
		}
		w := NewWriter(store)
		_, err = io.Copy(w, bytes.NewReader(randbytes.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		root, err := w.Close()
		if err != nil {
			t.Fatal(err)
		}
		r, err := NewReader(store, root)
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(&readbytes, r)
		if err != nil {
			t.Fatal(err)
		}
		expected := randbytes.Bytes()
		got := readbytes.Bytes()
		if len(expected) != len(got) {
			t.Fatalf("bad lengths %d != %d", len(expected), len(got))
		}
		for i := range expected {
			if expected[i] != got[i] {
				t.Fatalf("corrupt read at idx %d (%d != %d)", i, expected[i], got[i])
			}
		}
	}
}
