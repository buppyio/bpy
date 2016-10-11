package test

import (
	"bytes"
	"crypto/rand"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/remote/client"
	"github.com/buppyio/bpy/remote/server"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

type TestConn struct {
	pr *io.PipeReader
	pw *io.PipeWriter
}

func (conn *TestConn) Write(buf []byte) (int, error) { return conn.pw.Write(buf) }
func (conn *TestConn) Read(buf []byte) (int, error)  { return conn.pr.Read(buf) }
func (conn *TestConn) Close() error                  { conn.pr.Close(); conn.pw.Close(); return nil }

func newTestConnPair() (*TestConn, *TestConn) {
	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()
	conn1 := &TestConn{
		pr: pr1,
		pw: pw2,
	}
	conn2 := &TestConn{
		pr: pr2,
		pw: pw1,
	}
	return conn1, conn2
}

func TestRemotePacks(t *testing.T) {
	testPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testPath)
	cliConn, srvConn := newTestConnPair()
	go server.Serve(srvConn, testPath)
	c, err := client.Attach(cliConn, "abc")
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.NewPack("packs/testpack")
	if err != nil {
		t.Fatal(err)
	}
	nbytes := 1000000
	buf := make([]byte, nbytes, nbytes)
	_, err = io.ReadFull(rand.Reader, buf)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(p, bytes.NewBuffer(buf))
	if err != nil {
		t.Fatal(err)
	}
	err = p.Close()
	if err != nil {
		t.Fatal(err)
	}
	f, err := c.Open("packs/testpack")
	if err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(buf, data) {
		t.Fatal("data differs\n")
	}
	packs, err := remote.ListPacks(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 1 {
		t.Fatal("expected one pack")
	}
	if packs[0].Name != "testpack" {
		t.Fatal("incorrect pack name")
	}
	if packs[0].Size != uint64(len(buf)) {
		t.Fatal("incorrect pack size")
	}
}

func TestRoot(t *testing.T) {
	testPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testPath)
	key, err := bpy.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	cliConn, srvConn := newTestConnPair()
	go server.Serve(srvConn, testPath)
	c, err := client.Attach(cliConn, "abc")
	if err != nil {
		t.Fatal(err)
	}
	generation, err := remote.GetGeneration(c)
	if err != nil {
		t.Fatal(err)
	}

	_, ok, err := remote.GetRoot(c, &key)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected missing root")
	}

	root0 := [32]byte{}
	root0[0] = 1
	ok, err = remote.CasRef(c, &key, [32]byte{}, root0, generation)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected cas success")
	}

	val, ok, err := remote.GetRoot(c, &key)
	if !ok {
		t.Fatal("expected root")
	}
	if !reflect.DeepEqual(val, root0) {
		t.Fatal("bad val")
	}

	root1 := root0
	root1[0] = 2

	ok, err = remote.CasRef(c, &key, [32]byte{}, root1, generation)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected cas failure")
	}

	val, ok, err = remote.GetRoot(c, &key)
	if !ok {
		t.Fatal("expected root")
	}
	if !reflect.DeepEqual(val, root0) {
		t.Fatal("bad val")
	}

	ok, err = remote.CasRef(c, &key, root0, root1, generation)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected cas success")
	}

	val, ok, err = remote.GetRoot(c, &key)
	if !ok {
		t.Fatal("expected root")
	}
	if !reflect.DeepEqual(val, root1) {
		t.Fatal("bad val")
	}
}
