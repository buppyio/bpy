package cstore

import (
	"fmt"
	"github.com/buppyio/bpy/remote/client"
	"github.com/buppyio/bpy/remote/server"
	"github.com/buppyio/bpy/testhelp"
	"io"
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
	defer os.RemoveAll(d)
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

	clientcon, servercon := testhelp.NewTestConnPair()
	defer clientcon.Close()
	defer servercon.Close()
	go server.Serve(servercon, storepath)
	store, err := client.Attach(clientcon, "1234")
	if err != nil {
		t.Fatal(err)
	}
	testvals := make(map[[32]byte][]byte)
	key := [32]byte{}
	_, err = io.ReadFull(r, key[:])
	if err != nil {
		t.Fatal(err)
	}
	rdr, err := NewReader(store, key, cachepath)
	if err != nil {
		t.Fatal(err)
	}
	defer rdr.Close()
	for i := 0; i < 10; i++ {
		w, err := NewWriter(store, key, cachepath)
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < int(r.Int31())%500; j++ {
			nbytes := r.Int31() % 50
			rbytes := make([]byte, nbytes, nbytes)
			_, err = r.Read(rbytes)
			if err != nil {
				t.Fatal(err)
			}
			hash, err := w.Put(rbytes)
			if err != nil {
				t.Fatal(err)
			}
			gotv, err := w.Get(hash)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(rbytes, gotv) {
				t.Fatal(fmt.Errorf("values differ %v != %v", rbytes, gotv))
			}
			testvals[hash] = rbytes
		}

		err = w.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	for k, v := range testvals {
		gotv, err := rdr.Get(k)
		if err != nil {
			t.Fatal(err)
		}
		if len(v) == 0 && len(gotv) == 0 {
			continue
		}
		if !reflect.DeepEqual(v, gotv) {
			t.Fatal(fmt.Errorf("values differ %v != %v", v, gotv))
		}
	}
}
