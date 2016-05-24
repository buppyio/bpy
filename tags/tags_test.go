package tags

import (
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/proto9"
	"acha.ninja/bpy/remote"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
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

func TestTags(t *testing.T) {
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

	srv, err := remote.NewServer(servercon, storepath)
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve()

	remote, err := client9.NewClient(proto9.NewConn(clientcon, clientcon))
	if err != nil {
		t.Fatal(err)
	}
	err = remote.Attach("", "")
	if err != nil {
		t.Fatal(err)
	}

	testvals := make(map[string]string)
	testvals["foo"] = "bar"
	testvals["foo1"] = "bang"
	testvals["foo2"] = "baz"

	for k, v := range testvals {
		err = Create(remote, k, v)
		if err != nil {
			t.Fatal(err)
		}
	}

	tags, err := List(remote)
	if err != nil {
		t.Fatal(err)
	}

	if len(tags) != 3 {
		t.Fatal("incorrect number of tags")
	}

	if tags[0] != "foo" || tags[1] != "foo1" || tags[2] != "foo2" {
		t.Fatal("incorrect tag listing")
	}

	for k, v := range testvals {
		val, err := Get(remote, k)
		if err != nil {
			t.Fatal(err)
		}
		if val != v {
			t.Fatalf("value got('%s') != expected('%v')", v, val)
		}
	}

	err = Remove(remote, "foo", "...")
	if err == nil {
		t.Fatal("expected error")
	}
	err = Remove(remote, "foo", "bar")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Get(remote, "foo")
	if err == nil {
		t.Fatal("expecter error")
	}
	tags, err = List(remote)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Fatal("incorrect number of tags")
	}
}
