package drive

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestAttach(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	drive, err := Open(filepath.Join(testDir, "drive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer drive.Close()

	ok, err := drive.Attach("abc")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected ok attach")
	}

	ok, err = drive.Attach("abd")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected !ok attach")
	}

	ok, err = drive.Attach("abc")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected ok attach")
	}
}

func TestGCGeneration(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	drive, err := Open(filepath.Join(testDir, "drive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer drive.Close()

	gen, err := drive.GetGCGeneration()
	if err != nil {
		t.Fatal(err)
	}

	if gen != 0 {
		t.Fatal("unexpected gcGeneration")
	}

	gen, err = drive.StartGC()
	if err != nil {
		t.Fatal(err)
	}
	if gen != 1 {
		t.Fatal("unexpected gcGeneration")
	}

	err = drive.StopGC()
	if err != nil {
		t.Fatal(err)
	}

	gen, err = drive.GetGCGeneration()
	if err != nil {
		t.Fatal(err)
	}
	if gen != 2 {
		t.Fatal("unexpected gcGeneration")
	}
}

func TestCasRoot(t *testing.T) {
	testDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	drive, err := Open(filepath.Join(testDir, "drive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer drive.Close()

	root, version, err := drive.GetRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != "" || version != 0 {
		t.Fatal("unexpected root/version")
	}

	ok, err := drive.CasRoot("foo", 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected ok")
	}

	ok, err = drive.CasRoot("foo", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected ok")
	}

	ok, err = drive.CasRoot("foo", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected ok")
	}

	root, version, err = drive.GetRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != "" || version != 0 {
		t.Fatal("unexpected root/version")
	}

	ok, err = drive.CasRoot("foo", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("unexpected !ok")
	}

	root, version, err = drive.GetRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != "foo" || version != 1 {
		t.Fatal("unexpected root/version")
	}
}
