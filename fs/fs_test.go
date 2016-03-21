package fs

import (
	"acha.ninja/bpy/testhelp"
	"os"
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

	store := testhelp.NewMemStore()
	f := DirEnt{Name: "f", Size: 10, Mode: 0}

	hash, err := WriteDir(store, DirEnts{f})
	if err != nil {
		t.Fatal(err)
	}
	d := DirEnt{Name: "d", Size: 0, Mode: os.ModeDir, Data: hash}
	for i := 0; i < 3; i++ {
		hash, err = WriteDir(store, DirEnts{d})
		if err != nil {
			t.Fatal(err)
		}
		d.Data = hash
	}
	ent, err := Walk(store, hash, "/d/d/d/")
	if err != nil {
		t.Fatal(err)
	}
	if !ent.Mode.IsDir() {
		t.Fatal("expected dir")
	}
	ent, err = Walk(store, hash, "/d/d/d/f")
	if err != nil {
		t.Fatal(err)
	}
	if ent.Size != 10 {
		t.Fatal("bad size")
	}
}
