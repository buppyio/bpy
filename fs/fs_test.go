package fs

import (
	"acha.ninja/bpy/htree"
	"acha.ninja/bpy/testhelp"
	"math/rand"
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
	hash, err := WriteDir(store, dir, 0777)
	if err != nil {
		t.Fatal(err)
	}
	rdir, err := ReadDir(store, hash)
	if err != nil {
		t.Fatal(err)
	}
	if rdir[0].Name != "" {
		t.Fatal("missing current dir entry\n")
	}
	if !reflect.DeepEqual(dir, rdir[1:]) {
		t.Fatalf("dirs differ\n%v\n%v\n", dir, rdir)
	}
}

func TestWalk(t *testing.T) {

	store := testhelp.NewMemStore()
	f := DirEnt{Name: "f", Size: 10, Mode: 0}
	hash, err := WriteDir(store, DirEnts{f}, 0777)
	if err != nil {
		t.Fatal(err)
	}
	d := DirEnt{Name: "d", Size: 0, Mode: os.ModeDir, Data: hash}
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

func TestSeek(t *testing.T) {
	store := testhelp.NewMemStore()
	r := rand.New(rand.NewSource(3453))

	for n := 0; n < 10; n++ {
		nbytes := r.Int31() % 16
		data := make([]byte, nbytes, nbytes)
		r.Read(data)
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
				Name: "f",
				Mode: 0777,
				Size: int64(len(data)),
				Data: thash,
			}}, 0777)
		if err != nil {
			t.Fatal(err)
		}

		f, err := Open(store, dhash, "f")
		if err != nil {
			t.Fatal(err)
		}

		for i := 0; i < len(data); i++ {
			_, err = f.Seek(int64(i), 0)
			if err != nil {
				t.Fatal(err)
			}
			expected := data[i:]
			result := make([]byte, len(expected), len(expected))
			_, err = f.Read(result)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(expected, result) {
				t.Fatal("bad value")
			}
		}
		for i := 0; i < len(data); i++ {
			_, err = f.Seek(0, 0)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.Seek(int64(i), 1)
			if err != nil {
				t.Fatal(err)
			}
			expected := data[i:]
			result := make([]byte, len(expected), len(expected))
			_, err = f.Read(result)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(expected, result) {
				t.Fatal("bad value")
			}
		}
		for i := 0; i < len(data); i++ {
			_, err = f.Seek(-int64(i), 2)
			if err != nil {
				t.Fatal(err)
			}
			expected := data[len(data)-i:]
			result := make([]byte, len(expected), len(expected))
			_, err = f.Read(result)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(expected, result) {
				t.Fatal("bad value")
			}
		}
	}
}
