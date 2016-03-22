// Copyright (C) 2015  Andrew Chambers - andrewchamberss@gmail.com

package fsutil

import (
	"acha.ninja/bpy/testhelp"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
)

func TestStoreDirAndRestore(t *testing.T) {
	rd := rand.New(rand.NewSource(65464))
	for i := 0; i < 5; i++ {
		tmp, err := ioutil.TempDir("", "buppytestcpdir")
		if err != nil {
			t.Fatal(err)
		}
		randd := path.Join(tmp, "rand")
		restored := path.Join(tmp, "restored")
		err = os.Mkdir(randd, 0700)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmp)
		err = testhelp.RandomDirectoryTree(randd, testhelp.RandDirConfig{
			MaxDepth:    3,
			MaxSubdirs:  3,
			MaxFileSize: 1024 * 1024 * 1,
			MaxFiles:    3,
		}, rd)
		if err != nil {
			t.Fatal(err)
		}
		store := testhelp.NewMemStore()
		hash, err := CpHostDirToFs(store, randd)
		if err != nil {
			t.Fatal(err)
		}
		err = CpFsDirToHost(store, hash, restored)
		if err != nil {
			t.Fatal(err)
		}
		if !testhelp.DirEqual(randd, restored) {
			t.Fatalf("%s != %s", randd, restored)
		}
	}
}
