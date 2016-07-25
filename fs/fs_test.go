package fs

import (
	"acha.ninja/bpy/htree"
	"acha.ninja/bpy/testhelp"
	"io"
	"math/rand"
	"os"
	"reflect"
	"testing"
)

func TestDir(t *testing.T) {
	dir := DirEnts{
		{EntName: "Bar", EntSize: 4, EntMode: 5, EntModTime: 6, Data: [32]byte{1, 2, 3, 4}},
		{EntName: "Foo", EntSize: 0xffffff, EntMode: 0xffffff, EntModTime: 0xffff},
	}
	store := testhelp.NewMemStore()
	hash, err := WriteDir(store, dir, 0777)
	if err != nil {
		t.Fatal(err)
	}
	rdir, err := ReadDir(store, hash)
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
	hash, err := WriteDir(store, DirEnts{f}, 0777)
	if err != nil {
		t.Fatal(err)
	}
	d := DirEnt{EntName: "d", EntSize: 0, EntMode: os.ModeDir, Data: hash}
	for i := 0; i < 3; i++ {
		hash, err = WriteDir(store, DirEnts{d}, 0777)
		if err != nil {
			t.Fatal(err)
		}
		d.Data = hash
	}
	ent, err := Walk(store, hash, "/")
	if err != nil {
		t.Fatal(err)
	}
	if ent.Data != hash {
		t.Fatal("empty walk failed")
	}
	ent, err = Walk(store, hash, "")
	if err != nil {
		t.Fatal(err)
	}
	if ent.Data != hash {
		t.Fatal("empty walk failed")
	}
	ent, err = Walk(store, hash, "/d/d/d/")
	if err != nil {
		t.Fatal(err)
	}
	if !ent.EntMode.IsDir() {
		t.Fatal("expected dir")
	}
	ent, err = Walk(store, hash, "/d/d/d/f")
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
		dhash, err := WriteDir(store,
			DirEnts{DirEnt{
				EntName: "f",
				EntMode: 0777,
				EntSize: int64(len(data)),
				Data:    thash,
			}}, 0777)
		if err != nil {
			t.Fatal(err)
		}

		f, err := Open(store, dhash, "f")
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
	rdir, err := ReadDir(store, empty)
	if err != nil {
		t.Fatal(err)
	}
	ent := rdir[0]
	ent.EntName = "foo"
	notEmpty1, err := Insert(store, store, empty, "", ent)
	if err != nil {
		t.Fatal(err)
	}
	ent.EntName = "bar"
	notEmpty2, err := Insert(store, store, notEmpty1, "/foo/", ent)
	if err != nil {
		t.Fatal(err)
	}
	barEnt, err := Walk(store, notEmpty2, "/foo/bar/")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ent, barEnt) {
		t.Fatal("expected empty file")
	}
}
