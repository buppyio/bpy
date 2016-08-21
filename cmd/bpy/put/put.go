package put

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/fs/fsutil"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"path/filepath"
)

func Put() {
	refArg := flag.String("ref", "default", "ref put data into")
	flag.Parse()

	if len(flag.Args()) < 1 {
		common.Die("please specify the local folder to put into dest\n")
	}

	if len(flag.Args()) > 2 {
		common.Die("please specify the local folder and the dest folder\n")
	}

	srcPath, err := filepath.Abs(flag.Args()[0])
	if err != nil {
		common.Die("error getting src path: %s\n", err.Error())
	}

	destPath := "/"
	if len(flag.Args()) == 2 {
		destPath = flag.Args()[1]
	}

	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	generation, err := remote.GetGeneration(c)
	if err != nil {
		common.Die("error getting current gc generation: %s\n", err.Error())
	}

	store, err := common.GetCStore(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	ref, ok, err := remote.GetRef(c, &k, *refArg)
	if err != nil {
		common.Die("error fetching ref hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("ref '%s' does not exist\n", *refArg)
	}

	srcDirEnt, err := fsutil.CpHostToFs(store, srcPath)
	if err != nil {
		common.Die("error copying data: %s\n", err.Error())
	}

	newRootEnt, err := fs.Insert(store, ref.Root, destPath, srcDirEnt)
	if err != nil {
		common.Die("error inserting src into folder: %s\n", err.Error())
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}

	newRef := refs.Ref{
		Root: newRootEnt.HTree.Data,
	}

	ok, err = remote.CasRef(c, &k, *refArg, ref, newRef, generation)
	if err != nil {
		common.Die("creating ref: %s\n", err.Error())
	}

	if !ok {
		// XXX: loop here
		common.Die("ref concurrently modified, try again\n")
	}

}
