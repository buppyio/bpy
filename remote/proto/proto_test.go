package proto

import (
	"reflect"
	"testing"
)

func TestEncDec(t *testing.T) {
	buf := make([]byte, 1024, 1024)

	messages := []Message{
		&TError{
			Mid:     2,
			Message: "Error Message",
		},
		&TReadAt{
			Mid:    3,
			Fid:    4,
			Offset: 0xffffffffffffffff,
		},
		&RReadAt{
			Mid:  3,
			Data: []byte{1, 2, 3},
		},
	}

	for _, mIn := range messages {
		n, err := PackMessage(mIn, buf)
		if err != nil {
			t.Fatal(err)
		}
		mOut, err := UnpackMessage(buf[:n])
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(mIn, mOut) {
			t.Fatalf("%#v != %#v", mIn, mOut)
		}
	}
}
