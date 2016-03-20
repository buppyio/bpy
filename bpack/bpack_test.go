package bpack

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"reflect"
	"testing"
)

type bufwriteseeker struct {
	off int64
	buf []byte
}

func (b *bufwriteseeker) Seek(off int64, whence int) (int64, error) {
	if whence != 0 {
		panic("unexpected whence")
	}
	b.off = off
	return off, nil
}

func (b *bufwriteseeker) Write(buf []byte) (int, error) {
	for i := range buf {
		b.buf[int(b.off)+i] = buf[i]
	}
	b.off += int64(len(buf))
	return len(buf), nil
}

func TestBpack(t *testing.T) {
	var buf [1024 * 1024]byte

	w, err := NewWriter(&bufwriteseeker{off: 0, buf: buf[:]})
	if err != nil {
		t.Fatal(err)
	}
	rd := rand.New(rand.NewSource(76463))
	has := make(map[string][]byte)
	for i := 0; i < 1000; i++ {
		ksz := rd.Int31() % 100
		vsz := rd.Int31() % 100
		k := make([]byte, ksz, ksz)
		v := make([]byte, vsz, vsz)
		_, err = rd.Read(k)
		if err != nil {
			t.Fatal(err)
		}
		_, err = rd.Read(v)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := has[string(k)]
		if ok {
			continue
		}
		err = w.Add(string(k), v)
		has[string(k)] = v
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	r := NewReader(bytes.NewReader(buf[:]))
	err = r.ReadIndex()
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range has {
		gotv, ok, err := r.Get(string(k))
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("k=(%v) not found!\n", hex.EncodeToString([]byte(k)))
		}
		if !reflect.DeepEqual(v, gotv) {
			t.Fatalf("k=(%v) %v != %v", []byte(k), v, gotv)
		}
	}
}
