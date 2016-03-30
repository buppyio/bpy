package cstore

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCStore(t *testing.T) {
	r := rand.New(rand.NewSource(1234))
	d, err := ioutil.TempDir("", "cstoretest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(d)
	storepath := filepath.Join(d, "packs")
	cachepath := filepath.Join(d, "cache")
	err = os.MkdirAll(storepath, 0777)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(cachepath, 0777)
	if err != nil {
		t.Fatal(err)
	}
	testvals := make(map[[32]byte][]byte)
	for i := 0; i < 10; i++ {
		w, err := NewWriter(storepath, cachepath)
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < int(r.Int31())%500; j++ {
			nbytes := r.Int31() % 10
			rbytes := make([]byte, nbytes, nbytes)
			_, err = r.Read(rbytes)
			if err != nil {
				t.Fatal(err)
			}
			hash, err := w.Add(rbytes)
			if err != nil {
				t.Fatal(err)
			}
			testvals[hash] = rbytes
		}
		err = w.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	rdr, err := NewReader(storepath, cachepath)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range testvals {
		gotv, err := rdr.Get(k)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v, gotv) {
			t.Fatal(fmt.Errorf("values differ"))
		}
	}
}
