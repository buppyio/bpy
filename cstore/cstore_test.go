package cstore

import (
	"acha.ninja/bpy/testhelp"
	"testing"
)

func Testlru(t *testing.T) {
	a := testhelp.NewMemStore()
	b := testhelp.NewMemStore()
	c := testhelp.NewMemStore()

	fn := func(path string) error {
		switch path {
		case "a":
			return a, nil
		case "b":
			return b, nil
		case "c":
			return c, nil
		default:
			return nil, fmt.Errorf("no such path %s\n", err)
		}
	}
}
