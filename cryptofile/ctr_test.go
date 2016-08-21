package cryptofile

import (
	"reflect"
	"testing"
)

func TestCtr(t *testing.T) {
	c := newCtrState([]byte{0, 0, 0})

	c.Add(1)

	if !reflect.DeepEqual(c.Vec, []byte{0, 0, 1}) {
		t.Fatal("Add failed", c.Vec)
	}

	c.Add(0xffff)

	if !reflect.DeepEqual(c.Vec, []byte{1, 0, 0}) {
		t.Fatal("Add failed", c.Vec)
	}

	c.Add(0xffffff)

	if !reflect.DeepEqual(c.Vec, []byte{0, 0xff, 0xff}) {
		t.Fatal("Add failed", c.Vec)
	}

	xorVal := []byte{1, 0xff, 0}
	Xor(xorVal, c.Vec)

	if !reflect.DeepEqual(xorVal, []byte{1, 0, 0xff}) {
		t.Fatal("Add failed", c.Vec)
	}

	c.Reset()

	if !reflect.DeepEqual(c.Vec, []byte{0, 0, 0}) {
		t.Fatal("reset failed", c.Vec)
	}

}
