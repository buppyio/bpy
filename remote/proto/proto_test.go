package proto

import (
	"reflect"
	"testing"
)

func TestEncDec(t *testing.T) {
	buf := make([]byte, 1024, 1024)

	messages := []Message{
		&RError{
			Mid:     1,
			Message: "Error Message",
		},
		&TAttach{
			Mid:            2,
			Version:        "...",
			MaxMessageSize: 1234,
			KeyId:          "aaaaaaaaaaaaaaaaaaaaa",
		},
		&RAttach{
			Mid:            3,
			MaxMessageSize: 1234,
		},
		&TReadAt{
			Mid:    4,
			Fid:    5,
			Offset: 0xffffffffffffffff,
		},
		&RReadAt{
			Mid:  6,
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
