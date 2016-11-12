package test

import (
	"bytes"
	"crypto/rand"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/remote/client"
	"github.com/buppyio/bpy/remote/server"
	"github.com/buppyio/bpy/testhelp"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestRemotePacks(t *testing.T) {
	testPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testPath)
	cliConn, srvConn := testhelp.NewTestConnPair()
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
	cliConn, srvConn := testhelp.NewTestConnPair()
	go server.Serve(srvConn, testPath)
	c, err := client.Attach(cliConn, "abc")
	if err != nil {
		t.Fatal(err)
	}
	generation, err := remote.GetGeneration(c)
	if err != nil {
		t.Fatal(err)
	}

	_, version, ok, err := remote.GetRoot(c, &key)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected missing root")
	}

	root0 := [32]byte{}
	version0 := bpy.NextRootVersion(version)
	ok, err = remote.CasRoot(c, &key, root0, version0, generation)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected cas success")
	}

	val, version, ok, err := remote.GetRoot(c, &key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected root")
	}
	if !reflect.DeepEqual(val, root0) {
		t.Fatal("bad val")
	}
	if version != version0 {
		t.Fatal("bad version")
	}

	root1 := [32]byte{}
	root1[0] = 1

	ok, err = remote.CasRoot(c, &key, root1, version0, generation)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected cas failure")
	}

	val, version, ok, err = remote.GetRoot(c, &key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected root")
	}
	if !reflect.DeepEqual(val, root0) {
		t.Fatal("bad val")
	}
	if version != version0 {
		t.Fatal("bad version")
	}

	version1 := bpy.NextRootVersion(version0)
	ok, err = remote.CasRoot(c, &key, root1, version1, generation)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected cas success")
	}

	val, version, ok, err = remote.GetRoot(c, &key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected root")
	}
	if !reflect.DeepEqual(val, root1) {
		t.Fatal("bad val")
	}
	if version != version1 {
		t.Fatal("bad version")
	}
}
