// Copyright (C) 2015  Andrew Chambers - andrewchamberss@gmail.com

package archive

import (
	"acha.ninja/bpy/fs/fsutil"
	"acha.ninja/bpy/testhelp"
	"archive/tar"
	"archive/zip"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestTar(t *testing.T) {
	r := rand.New(rand.NewSource(1234))
	tmp, err := ioutil.TempDir("", "buppytesttar")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	randd := filepath.Join(tmp, "rand")
	err = os.Mkdir(randd, 0700)
	if err != nil {
		t.Fatal(err)
	}
	outtarpath := filepath.Join(tmp, "out.tar")
	outtar, err := os.Create(outtarpath)
	if err != nil {
		t.Fatal(err)
	}
	err = testhelp.RandomDirectoryTree(randd, testhelp.RandDirConfig{
		MaxDepth:    3,
		MaxSubdirs:  2,
		MaxFileSize: 1024 * 1024 * 1,
		MaxFiles:    3,
	}, r)
	if err != nil {
		t.Fatal(err)
	}
	store := testhelp.NewMemStore()
	dirEnt, err := fsutil.CpHostDirToFs(store, randd)
	if err != nil {
		t.Fatal(err)
	}
	err = Tar(store, dirEnt.Data, outtar)
	if err != nil {
		t.Fatal(err)
	}
	err = outtar.Close()
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(outtarpath)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(f)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		testfpath := filepath.Join(randd, h.Name)
		testf, err := os.Open(testfpath)
		if err != nil {
			t.Fatal(err)
		}
		d1, err := ioutil.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		d2, err := ioutil.ReadAll(testf)
		if err != nil {
			t.Fatal(err)
		}
		err = testf.Close()
		if err != nil {
			t.Fatal(err)
		}
		if len(d1) != len(d2) {
			t.Fatal("lengths differ")
		}
		for i := range d1 {
			if d1[i] != d2[i] {
				t.Fatalf("data at idx %d differs", i)
			}
		}
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = os.RemoveAll(tmp)
	if err != nil {
		t.Fatal(err)
	}
}

func TestZip(t *testing.T) {
	r := rand.New(rand.NewSource(1234))

	tmp, err := ioutil.TempDir("", "buppytestzip")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	randd := filepath.Join(tmp, "rand")
	err = os.Mkdir(randd, 0700)
	if err != nil {
		t.Fatal(err)
	}
	outzippath := filepath.Join(tmp, "out.zip")
	outzip, err := os.Create(outzippath)
	if err != nil {
		t.Fatal(err)
	}
	err = testhelp.RandomDirectoryTree(randd, testhelp.RandDirConfig{
		MaxDepth:    3,
		MaxSubdirs:  3,
		MaxFileSize: 1024 * 1024 * 1,
		MaxFiles:    3,
	}, r)
	if err != nil {
		t.Fatal(err)
	}
	store := testhelp.NewMemStore()
	dirEnt, err := fsutil.CpHostDirToFs(store, randd)
	if err != nil {
		t.Fatal(err)
	}
	err = Zip(store, dirEnt.Data, outzip)
	if err != nil {
		t.Fatal(err)
	}
	err = outzip.Close()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(outzippath)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		zr, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		testfpath := filepath.Join(randd, f.Name)
		testf, err := os.Open(testfpath)
		if err != nil {
			t.Fatal(err)
		}
		d1, err := ioutil.ReadAll(zr)
		if err != nil {
			t.Fatal(err)
		}
		d2, err := ioutil.ReadAll(testf)
		if err != nil {
			t.Fatal(err)
		}
		if len(d1) != len(d2) {
			t.Fatalf("lengths differ %d != %d", len(d1), len(d2))
		}
		for i := range d1 {
			if d1[i] != d2[i] {
				t.Fatalf("data at idx %d differs", i)
			}
		}
		err = zr.Close()
		if err != nil {
			t.Fatal(err)
		}
		err = testf.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	err = zr.Close()
	if err != nil {
		t.Fatal(err)
	}

}
