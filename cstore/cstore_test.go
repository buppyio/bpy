package cstore

import (
	"acha.ninja/bpy/remote/client"
	"acha.ninja/bpy/remote/server"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type testPipe struct {
	in  *io.PipeReader
	out *io.PipeWriter
}

func (p *testPipe) Read(buf []byte) (int, error)  { return p.in.Read(buf) }
func (p *testPipe) Write(buf []byte) (int, error) { return p.out.Write(buf) }
func (p *testPipe) Close() error                  { p.in.Close(); p.out.Close(); return nil }

func MakeConnection() (io.ReadWriteCloser, io.ReadWriteCloser) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	c1 := &testPipe{
		in:  r2,
		out: w1,
	}
	c2 := &testPipe{
		in:  r1,
		out: w2,
	}
	return c1, c2
}

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

	clientcon, servercon := MakeConnection()
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
	for i := 0; i < 100; i++ {
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
			testvals[hash] = rbytes
		}
		err = w.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	rdr, err := NewReader(store, key, cachepath)
	if err != nil {
		t.Fatal(err)
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
