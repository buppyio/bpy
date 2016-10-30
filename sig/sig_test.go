package sig

import (
	"github.com/buppyio/bpy"
	"testing"
)

func TestSigs(t *testing.T) {
	k1, err := bpy.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	k2, err := bpy.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	ver1 := uint64(0)
	ver2 := uint64(1)

	val1 := "a"
	val2 := "b"

	signed := SignValue(&k1, val1, ver1)

	if SignValue(&k1, val1, ver2) == signed {
		t.Fatal("signatures shouldn't match")
	}
	if SignValue(&k1, val2, ver1) == signed {
		t.Fatal("signatures shouldn't match")
	}
	if SignValue(&k2, val1, ver1) == signed {
		t.Fatal("signatures shouldn't match")
	}

}
