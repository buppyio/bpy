package cache

import (
	"crypto/sha256"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCache(t *testing.T) {
	r := rand.New(rand.NewSource(3453))
	tempDir, err := ioutil.TempDir("", "cachtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	maxSize := uint64(100)
	cachePath := filepath.Join(tempDir, "cache.db")

	cache, err := NewCache(cachePath, 0777, maxSize)
	if err != nil {
		t.Fatal(err)
	}

	hashes := [][32]byte{}
	for i := 0; i < 11; i++ {
		var buf [10]byte
		_, err := io.ReadFull(r, buf[:])
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(buf[:])
		err = cache.Put(hash, buf[:])
		if err != nil {
			t.Fatal(err)
		}
		val, ok, err := cache.Get(hash)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("unexpectedly not ok")
		}
		got := sha256.Sum256(val)
		if !reflect.DeepEqual(hash, got) {
			t.Fatal("bad value")
		}
		hashes = append(hashes, hash)
	}

	err = cache.Close()
	if err != nil {
		t.Fatal(err)
	}
	cache, err = NewCache(cachePath, 0777, maxSize)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	for idx, hash := range hashes {
		val, ok, err := cache.Get(hash)
		if err != nil {
			t.Fatal(err)
		}
		if idx == 0 {
			if ok {
				t.Fatal("unexpected ok")
			}
		} else {
			if !ok {
				t.Fatal("unexpectedly not ok")
			}
			got := sha256.Sum256(val)
			if !reflect.DeepEqual(hash, got) {
				t.Fatal("bad value")
			}
		}
	}
}

func BenchmarkCachePut(b *testing.B) {
	r := rand.New(rand.NewSource(3453))
	tempDir, err := ioutil.TempDir("", "cachtest")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	cachePath := filepath.Join(tempDir, "cache.db")
	cache, err := NewCache(cachePath, 0777, 50)
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf [10]byte
		_, err := io.ReadFull(r, buf[:])
		if err != nil {
			b.Fatal(err)
		}
		hash := sha256.Sum256(buf[:])
		err = cache.Put(hash, buf[:])
		if err != nil {
			b.Fatal(err)
		}
	}
}
