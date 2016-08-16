package test

import (
	"bytes"
	"crypto/rand"
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

func TestRemote(t *testing.T) {
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

func TestRefs(t *testing.T) {
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
	generation, err := remote.GetGeneration(c)
	if err != nil {
		t.Fatal(err)
	}

	testvals := make(map[string]string)
	testvals["foo"] = "bar"
	testvals["foo1"] = "bang"
	testvals["foo2"] = "baz"

	for k, v := range testvals {
		err = remote.NewRef(c, k, v, generation)
		if err != nil {
			t.Fatal(err)
		}
	}

	refs, err := remote.ListRefs(c)
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 3 {
		t.Fatal("incorrect number of refs")
	}

	if refs[0] != "foo" || refs[1] != "foo1" || refs[2] != "foo2" {
		t.Fatal("incorrect ref listing")
	}

	for k, v := range testvals {
		val, ok, err := remote.GetRef(c, k)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected ref")
		}
		if val != v {
			t.Fatalf("value got('%s') != expected('%v')", v, val)
		}
	}

	err = remote.RemoveRef(c, "foo", "...", generation)
	if err == nil {
		t.Fatal("expected error")
	}
	err = remote.RemoveRef(c, "foo", "bar", generation)
	if err != nil {
		t.Fatal(err)
	}
	_, ok, err := remote.GetRef(c, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected no ref")
	}
	refs, err = remote.ListRefs(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatal("incorrect number of refs")
	}

	ok, err = remote.CasRef(c, "foo2", "ba", "", generation)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected cas fail")
	}
	ok, err = remote.CasRef(c, "foo2", "baz", "casval", generation)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected cas success")
	}

	val, ok, err := remote.GetRef(c, "foo2")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected get success")
	}
	if val != "casval" {
		t.Fatal("bad val")
	}
}
