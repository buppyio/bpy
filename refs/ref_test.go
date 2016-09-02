package refs

import (
	"crypto/rand"
	"github.com/buppyio/bpy/testhelp"
	"io"
	"reflect"
	"testing"
)

func TestRefWithHist(t *testing.T) {
	store := testhelp.NewMemStore()

	ref := Ref{}
	_, err := io.ReadFull(rand.Reader, ref.Root[:])
	if err != nil {
		t.Fatal(err)
	}

	hash, err := PutRef(store, ref)
	if err != nil {
		t.Fatal(err)
	}

	got, err := GetRef(store, hash)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(ref, got) {
		t.Fatalf("%v != %v", ref, got)
	}
}

func TestRefNoHist(t *testing.T) {
	store := testhelp.NewMemStore()

	ref := Ref{}
	_, err := io.ReadFull(rand.Reader, ref.Root[:])
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadFull(rand.Reader, ref.Prev[:])
	if err != nil {
		t.Fatal(err)
	}
	ref.HasPrev = true

	hash, err := PutRef(store, ref)
	if err != nil {
		t.Fatal(err)
	}

	got, err := GetRef(store, hash)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(ref, got) {
		t.Fatalf("%v != %v", ref, got)
	}
}
