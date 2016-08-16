// Copyright (C) 2015  Andrew Chambers - andrewchamberss@gmail.com

package fsutil

import (
	"github.com/buppyio/bpy/testhelp"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
)

func TestStoreFile(t *testing.T) {
	// rd := rand.New(rand.NewSource(65464))
	tmp, err := ioutil.TempDir("", "buppytestcpdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	fileDir := path.Join(tmp, "randdir")
	restoredDir := path.Join(tmp, "restoreddir")
	err = os.MkdirAll(fileDir, 0777)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(restoredDir, 0777)
	if err != nil {
		t.Fatal(err)
	}
	randf := path.Join(fileDir, "randf")
	restored := path.Join(restoredDir, "randf")
	f, err := os.Create(randf)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	store := testhelp.NewMemStore()
	dirEnt, err := CpHostToFs(store, randf)
	if err != nil {
		t.Fatal(err)
	}

	err = hashTreeToHostFile(store, dirEnt.HTree.Data, restored, dirEnt.EntMode)
	if err != nil {
		t.Fatal(err)
	}

	if !testhelp.DirEqual(fileDir, restoredDir) {
		t.Fatalf("%s != %s", fileDir, restoredDir)
	}
}

func TestStoreDir(t *testing.T) {
	rd := rand.New(rand.NewSource(65464))
	for i := 0; i < 5; i++ {
		tmp, err := ioutil.TempDir("", "buppytestcpdir")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmp)
		randd := path.Join(tmp, "rand")
		restored := path.Join(tmp, "restored")
		err = os.Mkdir(randd, 0700)
		if err != nil {
			t.Fatal(err)
		}
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
		dirEnt, err := CpHostToFs(store, randd)
		if err != nil {
			t.Fatal(err)
		}
		err = CpFsToHost(store, dirEnt.HTree.Data, "/", restored)
		if err != nil {
			t.Fatal(err)
		}
		if !testhelp.DirEqual(randd, restored) {
			t.Fatalf("%s != %s", randd, restored)
		}
	}
}
