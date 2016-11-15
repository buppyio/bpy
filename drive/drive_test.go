package drive

import (
	"github.com/buppyio/bpy"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestEpoch(t *testing.T) {
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

	epoch1, err := drive.GetEpoch()
	if err != nil {
		t.Fatal(err)
	}

	epoch2, err := drive.StartGC()
	if err != nil {
		t.Fatal(err)
	}
	if epoch2 == epoch1 {
		t.Fatal("epoch did not increment")
	}

	err = drive.StopGC()
	if err != nil {
		t.Fatal(err)
	}

	epoch3, err := drive.GetEpoch()
	if err != nil {
		t.Fatal(err)
	}
	if epoch3 == epoch2 {
		t.Fatal("unexpected epoch")
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

	epoch, err := drive.GetEpoch()
	if err != nil {
		t.Fatal(err)
	}

	root, version, sig, err := drive.GetRoot()
	if err != nil {
		t.Fatal(err)
	}
	if root != "" || sig != "" {
		t.Fatal("unexpected root/sig")
	}

	ok, err := drive.CasRoot("foo", bpy.NextRootVersion(version), "sig", "bad epoch")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected ok")
	}

	ok, err = drive.CasRoot("foo", "", "sig", epoch)
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
	if root != "" || sig != "" {
		t.Fatal("unexpected root/sig")
	}

	ok, err = drive.CasRoot("foo", bpy.NextRootVersion(version), "sig", epoch)
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
	if root != "foo" || sig != "sig" {
		t.Fatalf("unexpected root/sig=%s/%s", root, sig)
	}
}

func TestUploadPack(t *testing.T) {
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

	err = drive.StartUpload("foobar")
	if err != nil {
		t.Fatal(err)
	}

	packs, err := drive.GetCompletePacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 0 {
		t.Fatal("expected 0 complete packs")
	}

	packs, err = drive.GetAllPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 1 || packs[0].Name != "foobar" {
		t.Fatal("expected one pack called foobar")
	}

	err = drive.FinishUpload("foobar", time.Now(), 0)
	if err != nil {
		t.Fatal(err)
	}

	err = drive.StartUpload("foobar")
	if err != ErrDuplicatePack {
		t.Fatal("expected duplicate pack error, got:", err)
	}

	packs, err = drive.GetCompletePacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 1 || packs[0].Name != "foobar" {
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

	epoch, err := drive.GetEpoch()
	if err != nil {
		t.Fatal(err)
	}

	err = drive.StartUpload("foobar")
	if err != nil {
		t.Fatal(err)
	}
	err = drive.FinishUpload("foobar", time.Now(), 0)
	if err != nil {
		t.Fatal(err)
	}

	err = drive.StartUpload("foobarbaz")
	if err != nil {
		t.Fatal(err)
	}
	err = drive.FinishUpload("foobarbaz", time.Now(), 0)
	if err != nil {
		t.Fatal(err)
	}

	err = drive.RemovePack("foobar", epoch)
	if err != ErrGCNotRunning {
		t.Fatal("expected ErrGCNotRunning error, got: ", err)
	}

	packs, err := drive.GetAllPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 2 {
		t.Fatal("expected remove failed")
	}

	epoch, err = drive.StartGC()
	if err != nil {
		t.Fatal(err)
	}

	err = drive.RemovePack("foobar", epoch)
	if err != nil {
		t.Fatal(err)
	}

	err = drive.StopGC()
	if err != nil {
		t.Fatal(err)
	}

	packs, err = drive.GetAllPacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 1 || packs[0].Name != "foobarbaz" {
		t.Fatal("expected remove success")
	}

}
