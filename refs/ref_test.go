package refs

import (
	"github.com/buppyio/bpy"
	"reflect"
	"testing"
)

func TestKeySigs(t *testing.T) {
	k, err := bpy.NewKey()
	if err != nil {
		t.Fatal(err)
	}

	r := Ref{}
	signed, err := SerializeAndSign(&k, r)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ParseRef(&k, signed)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r, got) {
		t.Fatal("parsing ref failed")
	}

	for i := 0; i < len(signed); i++ {
		corrupt := []byte(signed)
		corrupt[i] = corrupt[i] + 1
		_, err := ParseRef(&k, string(corrupt))
		if err == nil {
			t.Fatal("expected failure")
		}
	}
}
