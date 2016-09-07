package refs

import (
	"crypto/rand"
	"github.com/buppyio/bpy/testhelp"
	"io"
	"reflect"
	"testing"
	"time"
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

func TestGetAt(t *testing.T) {
	store := testhelp.NewMemStore()

	ref := Ref{}
	ref.CreatedAt = 0
	ref.HasPrev = false

	hash, err := PutRef(store, ref)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i < 10; i++ {
		ref.Prev = hash
		ref.HasPrev = true
		ref.CreatedAt = int64(i)
		ref.Root[0] = byte(i)
		hash, err = PutRef(store, ref)
		if err != nil {
			t.Fatal(err)
		}
	}

	got, ok, err := GetAtTime(store, ref, time.Unix(5, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Root[0] != 5 {
		t.Fatal("incorrect root value")
	}

	got, ok, err = GetAtTime(store, ref, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Root[0] != 9 {
		t.Fatal("incorrect root value", got.Root[0])
	}

}

func TestGetNVersionsAgo(t *testing.T) {
	store := testhelp.NewMemStore()

	ref := Ref{}
	ref.CreatedAt = 0
	ref.HasPrev = false

	hash, err := PutRef(store, ref)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i < 10; i++ {
		ref.Prev = hash
		ref.HasPrev = true
		ref.CreatedAt = int64(i)
		ref.Root[0] = byte(i)
		hash, err = PutRef(store, ref)
		if err != nil {
			t.Fatal(err)
		}
	}

	got, err := GetNVersionsAgo(store, ref, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root[0] != 9 {
		t.Fatal("incorrect root value")
	}

	got, err = GetNVersionsAgo(store, ref, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root[0] != 4 {
		t.Fatal("incorrect root value", got.Root[0])
	}

	got, err = GetNVersionsAgo(store, ref, 100)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root[0] != 0 {
		t.Fatal("incorrect root value", got.Root[0])
	}
}
