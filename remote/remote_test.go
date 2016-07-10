package remote

import (
	"acha.ninja/bpy/remote/client"
	"acha.ninja/bpy/remote/server"
	"io"
	"io/ioutil"
	"os"
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
	defer os.Remove(testPath)
	cliConn, srvConn := newTestConnPair()
	go server.Serve(srvConn, testPath)
	c, err := client.Attach(cliConn, "abc")
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.NewPack()
	if err != nil {
		t.Fatal(err)
	}
	err = p.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = c.Close()
	if err != nil {
		t.Fatal(err)
	}
}
