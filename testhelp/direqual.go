package testhelp

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
)

func DirEqual(l, r string) bool {
	ld, err := ioutil.ReadDir(l)
	if err != nil {
		panic(err)
	}
	rd, err := ioutil.ReadDir(r)
	if err != nil {
		panic(err)
	}
	if len(ld) != len(rd) {
		return false
	}
	for idx := range ld {
		if ld[idx].Mode() != rd[idx].Mode() {
			return false
		}
		if ld[idx].Mode().IsDir() {
			if !DirEqual(filepath.Join(l, ld[idx].Name()), filepath.Join(r, rd[idx].Name())) {
				return false
			}
		} else {
			d1, err := ioutil.ReadFile(filepath.Join(l, ld[idx].Name()))
			if err != nil {
				panic(err)
			}
			d2, err := ioutil.ReadFile(filepath.Join(r, rd[idx].Name()))
			if err != nil {
				panic(err)
			}
			if bytes.Compare(d1, d2) != 0 {
				return false
			}
		}
	}
	return true
}
