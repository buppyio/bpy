package cstore

import (
	"github.com/buppyio/bpy/testhelp"
	"io"
	"math/rand"
	"reflect"
	"testing"
)

func TestMemCachedStore(t *testing.T) {
	reference := testhelp.NewMemStore()
	maxCacheSize := uint64(100)
	cached := NewMemCachedCStore(testhelp.NewMemStore(), maxCacheSize)
	r := rand.New(rand.NewSource(3453))

	hashes := [][32]byte{}
	for i := 0; i < 100; i++ {
		var buf [10]byte
		_, err := io.ReadFull(r, buf[:])
		if err != nil {
			t.Fatal(err)
		}
		h1, err := reference.Put(buf[:])
		if err != nil {
			t.Fatal(err)
		}
		h2, err := cached.Put(buf[:])
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(h1, h2) {
			t.Fatal("hashes differ")
		}
		hashes = append(hashes, h1)
	}
	for i := 0; i < 100000; i++ {
		hash := hashes[r.Int()%len(hashes)]

		v1, err := reference.Get(hash)
		if err != nil {
			t.Fatal(err)
		}
		v2, err := cached.Get(hash)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v1, v2) {
			t.Fatal("values differ")
		}
	}
	err := reference.Flush()
	if err != nil {
		t.Fatal(err)
	}
	err = cached.Flush()
	if err != nil {
		t.Fatal(err)
	}

}
