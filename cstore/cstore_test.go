package cstore

import (
	"acha.ninja/bpy/testhelp"
	"testing"
	"io/ioutil"
	"math/rand"
)

func TestCStore(t *testing.T) {
	r := rand.New(rand.NewSource(1234))
	d, err := ioutil.TempDir("")
	if err != nil {
		t.Fatal(d)
	}
	defer os.Remove(d)
	cachepath := filepath.Join(d, "cache")
	storepath := filepath.Join(d, "packs")
	testvals := make(map[string][]byte)
	for i := 0; i < 10; i++ {
		w, err := NewWriter(storepath, cachepath)
		if err != nil {
			t.Fatal(d)
		}
		for j := 0; j < 100 ; j++ {
			nbytes := r.RandInt31() % 10
			rbytes := make([]byte, nbytes, nbytes)
			_, err = r.Read(rbytes)
			if err != nil {
				t.Fatal(d)
			}
			hash, err = w.Add(rbytes)
			if err != nil {
				t.Fatal(d)
			}
			testvals[string(hash[:])] = rbytes
		}
		err = w.Close()
		if err != nil {
			t.Fatal(d)
		}
	}
}
