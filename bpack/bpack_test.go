package bpack

import (
	"bytes"
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
	w.Add("", []byte("b"))
	w.Add("c", []byte(""))
	w.Add("test", []byte("vector"))
	w.Add("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", []byte("zzz"))
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	r := NewReader(bytes.NewReader(buf[:]))
	err = r.ReadIndex()
	if err != nil {
		t.Fatal(err)
	}
	v, _, err := r.Get("")
	if string(v) != "b" || err != nil {
		t.Fatal("Get failed", v)
	}
	v, _, err = r.Get("c")
	if string(v) != "" || err != nil {
		t.Fatal("Get failed", v)
	}
	v, _, err = r.Get("test")
	if string(v) != "vector" || err != nil {
		t.Fatal("Get failed", v)
	}
	v, _, err = r.Get("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	if string(v) != "zzz" || err != nil {
		t.Fatal("Get failed", v)
	}
	v, ok, err := r.Get("nothing")
	if err != nil {
		t.Fatal("Get failed", err)
	}
	if ok == true {
		t.Fatal("Get succeeded?", v)
	}
}
