package remote

import (
	"acha.ninja/bpy/remote/client"
	"acha.ninja/bpy/remote/server"
	"bytes"
	"crypto/rand"
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
	packs, err := c.ListPacks()
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
