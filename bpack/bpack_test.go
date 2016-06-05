package bpack

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
)

type bufwriter struct {
	off uint64
	buf []byte
}

func (b *bufwriter) Write(buf []byte) (int, error) {
	for i := range buf {
		b.buf[int(b.off)+i] = buf[i]
	}
	b.off += uint64(len(buf))
	return len(buf), nil
}

func (b *bufwriter) Close() error {
	return nil
}

type bufreader struct {
	buf *bytes.Reader
}

func (b *bufreader) Seek(off int64, whence int) (int64, error) {
	return b.buf.Seek(off, whence)
}

func (b *bufreader) Read(buf []byte) (int, error) {
	return b.buf.Read(buf)
}

func (b *bufreader) Close() error {
	return nil
}

func TestBpack(t *testing.T) {
	var buf [1024 * 1024 * 10]byte

	bufw := &bufwriter{off: 0, buf: buf[:]}
	w, err := NewWriter(bufw)
	if err != nil {
		t.Fatal(err)
	}
	rd := rand.New(rand.NewSource(76463))
	has := make(map[string][]byte)
	for i := 0; i < 10000; i++ {
		ksz := rd.Int31() % 10
		vsz := rd.Int31() % 10
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
	_, err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	r := NewReader(&bufreader{buf: bytes.NewReader(buf[:bufw.off])}, bufw.off)
	err = r.ReadIndex()
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range has {
		gotv, err := r.Get(string(k))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v, gotv) {
			t.Fatalf("k=(%v) %v != %v", []byte(k), v, gotv)
		}
	}
}

func TestEncrryptedBpack(t *testing.T) {
	var buf [1024 * 1024 * 10]byte

	bufw := &bufwriter{off: 0, buf: buf[:]}
	w, err := NewEncryptedWriter(bufw, [32]byte{})
	if err != nil {
		t.Fatal(err)
	}
	rd := rand.New(rand.NewSource(76463))
	has := make(map[string][]byte)
	for i := 0; i < 10000; i++ {
		ksz := rd.Int31() % 10
		vsz := rd.Int31() % 10
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
	_, err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewEncryptedReader(&bufreader{buf: bytes.NewReader(buf[:bufw.off])}, [32]byte{}, int64(bufw.off))
	if err != nil {
		t.Fatal(err)
	}
	err = r.ReadIndex()
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range has {
		gotv, err := r.Get(string(k))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v, gotv) {
			t.Fatalf("k=(%v) %v != %v", []byte(k), v, gotv)
		}
	}
}
