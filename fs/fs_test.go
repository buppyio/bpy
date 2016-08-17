package fs

import (
	"github.com/buppyio/bpy/htree"
	"github.com/buppyio/bpy/testhelp"
	"io"
	"math/rand"
	"os"
	"reflect"
	"testing"
)

func TestDir(t *testing.T) {
	dir := DirEnts{
		{EntName: "Bar", EntSize: 4, EntMode: 5, EntModTime: 6},
		{EntName: "Foo", EntSize: 0xffffff, EntMode: 0xffffff, EntModTime: 0xffff},
	}
	store := testhelp.NewMemStore()
	dirEnt, err := WriteDir(store, dir, 0777)
	if err != nil {
		t.Fatal(err)
	}
	rdir, err := ReadDir(store, dirEnt.HTree.Data)
	if err != nil {
		t.Fatal(err)
	}
	if rdir[0].EntName != "." {
		t.Fatal("missing current dir entry\n")
	}
	if !reflect.DeepEqual(dir, rdir[1:]) {
		t.Fatalf("dirs differ\n%v\n%v\n", dir, rdir)
	}
}

func TestWalk(t *testing.T) {
	store := testhelp.NewMemStore()
	f := DirEnt{EntName: "f", EntSize: 10, EntMode: 0}
	dirEnt, err := WriteDir(store, DirEnts{f}, 0777)
	if err != nil {
		t.Fatal(err)
	}
	d := DirEnt{EntName: "d", EntSize: 0, EntMode: os.ModeDir, HTree: dirEnt.HTree}
	for i := 0; i < 3; i++ {
		dirEnt, err = WriteDir(store, DirEnts{d}, 0777)
		if err != nil {
			t.Fatal(err)
		}
		d.HTree = dirEnt.HTree
	}
	ent, err := Walk(store, dirEnt.HTree.Data, "/")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ent.HTree, dirEnt.HTree) {
		t.Fatalf("empty walk failed %v != %v", ent.HTree, dirEnt.HTree)
	}
	ent, err = Walk(store, dirEnt.HTree.Data, "")
	if err != nil {
		t.Fatal(err)
	}
	if ent.HTree != dirEnt.HTree {
		t.Fatal("empty walk failed")
	}
	ent, err = Walk(store, dirEnt.HTree.Data, "/d/d/d/")
	if err != nil {
		t.Fatal(err)
	}
	if !ent.EntMode.IsDir() {
		t.Fatal("expected dir")
	}
	ent, err = Walk(store, dirEnt.HTree.Data, "/d/d/d/f")
	if err != nil {
		t.Fatal(err)
	}
	if ent.EntSize != 10 {
		t.Fatal("bad size")
	}
}

func TestSeek(t *testing.T) {
	store := testhelp.NewMemStore()
	r := rand.New(rand.NewSource(3453))

	for n := 0; n < 10; n++ {
		nbytes := r.Int31() % 16
		data := make([]byte, nbytes, nbytes)
		io.ReadFull(r, data)
		tw := htree.NewWriter(store)
		_, err := tw.Write(data)
		if err != nil {
			t.Fatal(err)
		}
		thash, err := tw.Close()
		if err != nil {
			t.Fatal(err)
		}
		dirEnt, err := WriteDir(store,
			DirEnts{DirEnt{
				EntName: "f",
				EntMode: 0777,
				EntSize: int64(len(data)),
				HTree:   thash,
			}}, 0777)
		if err != nil {
			t.Fatal(err)
		}

		f, err := Open(store, dirEnt.HTree.Data, "f")
		if err != nil {
			t.Fatal(err)
		}

		for i := 0; i < len(data); i++ {
			_, err = f.Seek(int64(i), io.SeekStart)
			if err != nil {
				t.Fatal(err)
			}
			expected := data[i:]
			result := make([]byte, len(expected), len(expected))
			_, err = io.ReadFull(f, result)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(expected, result) {
				t.Fatal("bad value")
			}
		}
		for i := 0; i < len(data); i++ {
			_, err = f.Seek(0, io.SeekStart)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.Seek(int64(i), io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			expected := data[i:]
			result := make([]byte, len(expected), len(expected))
			_, err = io.ReadFull(f, result)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(expected, result) {
				t.Fatal("bad value")
			}
		}
		for i := 0; i < len(data); i++ {
			_, err = f.Seek(-int64(i), io.SeekEnd)
			if err != nil {
				t.Fatal(err)
			}
			expected := data[len(data)-i:]
			result := make([]byte, len(expected), len(expected))
			_, err = io.ReadFull(f, result)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(expected, result) {
				t.Fatal("bad value")
			}
		}
	}
}

func TestInsert(t *testing.T) {
	store := testhelp.NewMemStore()
	empty, err := EmptyDir(store, 0755)
	if err != nil {
		t.Fatal(err)
	}
	rdir, err := ReadDir(store, empty.HTree.Data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rdir) != 1 {
		t.Fatal("expected empty dir")
	}
	ent := rdir[0]
	ent.EntName = "foo"
	notEmpty1, err := Insert(store, empty.HTree.Data, "", ent)
	if err != nil {
		t.Fatal(err)
	}
	rdir, err = ReadDir(store, notEmpty1.HTree.Data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rdir) != 2 {
		t.Fatal("expected single folder")
	}
	notEmpty2, err := Insert(store, notEmpty1.HTree.Data, "/foo/bar", ent)
	if err != nil {
		t.Fatal(err)
	}
	barEnt, err := Walk(store, notEmpty2.HTree.Data, "/foo/bar/")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ent.HTree, barEnt.HTree) {
		t.Fatal("expected empty file", ent, barEnt)
	}
}

func TestRemove(t *testing.T) {
	store := testhelp.NewMemStore()
	empty, err := EmptyDir(store, 0755)
	if err != nil {
		t.Fatal(err)
	}
	rdir, err := ReadDir(store, empty.HTree.Data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rdir) != 1 {
		t.Fatal("expected empty dir")
	}
	ent := rdir[0]
	ent.EntName = "foo"
	notEmpty1, err := Insert(store, empty.HTree.Data, "", ent)
	if err != nil {
		t.Fatal(err)
	}
	withFooRemoved, err := Remove(store, notEmpty1.HTree.Data, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(empty, withFooRemoved) {
		t.Fatal("expected empty file")
	}
}

func TestCopy(t *testing.T) {
	store := testhelp.NewMemStore()
	empty, err := EmptyDir(store, 0755)
	if err != nil {
		t.Fatal(err)
	}
	notEmpty1, err := Copy(store, empty.HTree.Data, "/foo", "/")
	if err != nil {
		t.Fatal(err)
	}
	walkEnt, err := Walk(store, notEmpty1.HTree.Data, "/foo")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(empty.HTree, walkEnt.HTree) {
		t.Fatal("expected empty folder")
	}
}

func TestMove(t *testing.T) {
	store := testhelp.NewMemStore()
	empty, err := EmptyDir(store, 0755)
	if err != nil {
		t.Fatal(err)
	}
	notEmpty1, err := Copy(store, empty.HTree.Data, "/bar", "/")
	if err != nil {
		t.Fatal(err)
	}
	moveDir, err := Move(store, notEmpty1.HTree.Data, "/bang", "/bar")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Walk(store, moveDir.HTree.Data, "/bar")
	if err == nil {
		t.Fatal("expected error")
	}
	walkEnt, err := Walk(store, moveDir.HTree.Data, "/bang")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(empty.HTree, walkEnt.HTree) {
		t.Fatal("expected empty folder")
	}
}
