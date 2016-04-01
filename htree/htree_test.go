package htree

import (
	"acha.ninja/bpy/testhelp"
	"bytes"
	"io"
	"math/rand"
	"testing"
)

func TestHTree(t *testing.T) {
	for i := 0; i < 25; i++ {
		var randbytes bytes.Buffer
		var readbytes bytes.Buffer

		store := testhelp.NewMemStore()
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

		_, err = r.Seek(int64(len(expected)))
		if err != io.EOF {
			t.Fatal("Seek should hit EOF")
		}

		for i := 0; i < 100; i++ {
			seekto := int64(rand.Int31()) % int64(len(expected))
			seekedto, err := r.Seek(seekto)
			if err != nil {
				t.Fatal("Seek failed %s", err.Error())
			}
			if seekedto != seekto {
				t.Fatal("Seek returned bad offset")
			}
			seekbuf := []byte{0}
			_, err = r.Read(seekbuf)
			if err != nil {
				t.Fatal("Seek failed %s", err.Error())
			}
			if seekbuf[seekto] != expected[0] {
				t.Fatal("Seek gave wrong value differ")
			}
		}
	}
}
