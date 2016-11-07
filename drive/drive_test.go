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

	root, version, sig, err := drive.GetRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != "" || sig != "" || version != 0 {
		t.Fatal("unexpected root/version/sig")
	}

	ok, err := drive.CasRoot("foo", 2, "sig", 2)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected ok")
	}

	ok, err = drive.CasRoot("foo", 1, "sig", 2)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected ok")
	}

	ok, err = drive.CasRoot("foo", 2, "sig", 0)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected ok")
	}

	root, version, sig, err = drive.GetRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != "" || sig != "" || version != 0 {
		t.Fatal("unexpected root/version/sig")
	}

	ok, err = drive.CasRoot("foo", 1, "sig", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("unexpected !ok")
	}

	root, version, sig, err = drive.GetRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != "foo" || version != 1 || sig != "sig" {
		t.Fatal("unexpected root/version/sig")
	}
}

func TestAddPack(t *testing.T) {
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

	err = drive.AddPack("foobar", 5)
	if err != ErrGCOccurred {
		t.Fatal(err)
	}

	err = drive.AddPack("foobar", 0)
	if err != nil {
		t.Fatal(err)
	}

	err = drive.AddPack("foobar", 0)
	if err != ErrDuplicatePack {
		t.Fatal("expected duplicate pack error, got:", err)
	}

	packs, err := drive.GetPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 1 || packs[0] != "foobar" {
		t.Fatal("expected one pack called foobar")
	}

}

func TestRemovePack(t *testing.T) {
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

	err = drive.AddPack("foobar", 0)
	if err != nil {
		t.Fatal(err)
	}
	err = drive.AddPack("foobarbaz", 0)
	if err != nil {
		t.Fatal(err)
	}

	err = drive.RemovePack("foobar", 1)
	if err != ErrGCOccurred {
		t.Fatal("expected GCOccurred error, got: ", err)
	}

	packs, err := drive.GetPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 2 {
		t.Fatal("expected remove failed")
	}

	err = drive.RemovePack("foobar", 0)
	if err != nil {
		t.Fatal(err)
	}

	packs, err = drive.GetPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 1 || packs[0] != "foobarbaz" {
		t.Fatal("expected remove success")
	}

}
