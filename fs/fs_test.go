package fs

import (
	"acha.ninja/bpy/testhelp"
	"reflect"
	"testing"
)

func TestDir(t *testing.T) {
	dir := DirEnts{
		{Name: "Bar", Size: 4, Mode: 5, ModTime: 6, Data: [32]byte{1, 2, 3, 4}},
		{Name: "Foo", Size: 0xffffff, Mode: 0xffffff, ModTime: 0xffff},
	}
	store := testhelp.NewMemStore()
	hash, err := WriteDir(store, dir)
	if err != nil {
		t.Fatal(err)
	}
	rdir, err := ReadDir(store, hash)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dir, rdir) {
		t.Fatalf("dirs differ\n%v\n%v\n", dir, rdir)
	}
}

func TestWalk(t *testing.T) {
	t.Fatal("unimplemented test")
}
