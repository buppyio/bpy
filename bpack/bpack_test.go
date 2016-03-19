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
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	r := NewReader(bytes.NewReader(buf[:]))
	err = r.ReadIndex()
	if err != nil {
		t.Fatal(err)
	}
	v, _, _ := r.Get("")
	if string(v) != "b" {
		t.Fatal("Get failed", v)
	}
	v, _, _ = r.Get("c")
	if string(v) != "" {
		t.Fatal("Get failed", v)
	}
	v, _, _ = r.Get("test")
	if string(v) != "vector" {
		t.Fatal("Get failed", v)
	}
	_, ok, _ := r.Get("nothing")
	if ok != false {
		t.Fatal("Get succeeded?")
	}
}
