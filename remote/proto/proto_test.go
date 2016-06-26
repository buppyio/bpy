package proto

import (
	"reflect"
	"testing"
)

func TestErrorEncDec(t *testing.T) {
	buf := make([]byte, 1024, 1024)
	mIn := &TError{
		Mid:     2,
		Message: "Error Message",
	}
	n, err := PackMessage(mIn, buf)
	if err != nil {
		t.Fatal(err)
	}
	mOut, err := UnpackMessage(buf[:n])
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(mIn, mOut) {
		t.Fatalf("%v != %v", mIn, mOut)
	}
}
