package cstore

import (
	"github.com/buppyio/bpy/cstore/cache"
	"github.com/buppyio/bpy/testhelp"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestMemCachedStore(t *testing.T) {
	reference := testhelp.NewMemStore()

	tempDir, err := ioutil.TempDir("", "cachtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cacheDBPath := filepath.Join(tempDir, "cache.db")
	server, err := cache.NewServer(cacheDBPath, 0777, 100)
	if err != nil {
		t.Fatal(err)
	}

	con1, con2 := testhelp.NewTestConnPair()
	defer con1.Close()
	defer con2.Close()

	go server.ServeConn(con1)
	client, err := cache.NewClient(con2)
	if err != nil {
		t.Fatal(err)
	}

	cached := NewCachedCStore(testhelp.NewMemStore(), client)
	r := rand.New(rand.NewSource(time.Now().Unix()))

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
	for i := 0; i < 1000; i++ {
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
	err = cached.Close()
	if err != nil {
		t.Fatal(err)
	}
}
