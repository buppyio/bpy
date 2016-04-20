package bpy

import (
	"encoding/hex"
	"reflect"
	"testing"
	"testing/quick"
)

func TestParseHash(t *testing.T) {
	fn := func(randbuf [32]byte) bool {
		str := hex.EncodeToString(randbuf[:])
		buf, err := ParseHash(str)
		if err != nil {
			return false
		}
		if !reflect.DeepEqual(buf, randbuf) {
			return false
		}
		return true
	}
	err := quick.Check(fn, nil)
	if err != nil {
		t.Fatalf("Fail: %s", err.Error())
	}
	_, err = ParseHash("")
	if err == nil {
		t.Fatalf("expected an error\n")
	}
	_, err = ParseHash("16ecab1875791e2b6ed0c9a6dae5a12a79d92120e1c3afbd3a9c8535ce44666d")
	if err != nil {
		t.Fatalf("expected an error\n")
	}
	_, err = ParseHash("16ecab1875791e2b6ed0c9a6dae5a12a79d92120e1c3afbd3a9c8535ce44666")
	if err == nil {
		t.Fatalf("expected an error\n")
	}
	_, err = ParseHash("16ecab1875791e2b6ed0c9a6dae5a12a79d92120e1c3afbd3a9c8535ce44666x")
	if err == nil {
		t.Fatalf("expected an error\n")
	}
}
