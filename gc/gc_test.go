package gc

import (
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cstore"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/fs/fsutil"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/remote/client"
	"github.com/buppyio/bpy/remote/server"
	"github.com/buppyio/bpy/testhelp"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestGarbageCollection(t *testing.T) {
	testPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testPath)
	cachePath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cachePath)

	k, err := bpy.NewKey()
	if err != nil {
		t.Fatal(err)
	}

	cliConn, srvConn := testhelp.NewTestConnPair()
	go server.Serve(srvConn, testPath)

	c, err := client.Attach(cliConn, "keyid")
	if err != nil {
		t.Fatal(err)
	}

	generation, err := remote.GetGeneration(c)
	if err != nil {
		t.Fatalf("error getting current gc generation: %s\n", err.Error())
	}

	store, err := cstore.NewWriter(c, k.CipherKey, cachePath)
	if err != nil {
		t.Fatalf("error creating cstore: %s\n", err.Error())
	}

	input := filepath.Join(os.Getenv("GOPATH"), "src/github.com/buppyio/bpy")

	_, err = fsutil.CpHostToFs(store, input)
	if err != nil {
		t.Fatalf("copying test dir failed: %s\n", err)
	}

	emptyDir, err := fs.EmptyDir(store, 0777)
	if err != nil {
		t.Fatalf("writing empty dir failed: %s\n", err)
	}

	hash, err := refs.PutRef(store, refs.Ref{Root: emptyDir.HTree.Data})
	if err != nil {
		t.Fatalf("writing ref failed: %s\n", err)
	}

	err = store.Flush()
	if err != nil {
		t.Fatalf("error flushing ref: %s\n", err)
	}

	ok, err := remote.CasRoot(c, &k, hash, 1, generation)
	if err != nil {
		t.Fatalf("error swapping root: %s\n", err)
	}

	if !ok {
		t.Fatalf("error swapping root\n")
	}

	err = GC(c, store, nil, &k)
	if err != nil {
		t.Fatalf("gc failed: %s\n", err)
	}
}
