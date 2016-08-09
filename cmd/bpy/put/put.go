package put

import (
	"encoding/hex"
	"flag"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/fs/fsutil"
	"github.com/buppyio/bpy/remote"
	"path/filepath"
)

func Put() {
	tagArg := flag.String("tag", "default", "tag put data into")
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

	store, err := common.GetCStore(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	generation, err := remote.GetGeneration(c)
	if err != nil {
		common.Die("error getting current gc generation: %s\n", err.Error())
	}

	tagHash, ok, err := remote.GetTag(c, *tagArg)
	if err != nil {
		common.Die("error fetching tag hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("tag '%s' does not exist\n", *tagArg)
	}

	destHash, err := bpy.ParseHash(tagHash)
	if err != nil {
		common.Die("error parsing hash: %s\n", err.Error())
	}

	srcDirEnt, err := fsutil.CpHostToFs(store, srcPath)
	if err != nil {
		common.Die("error copying data: %s\n", err.Error())
	}

	newRoot, err := fs.Insert(store, destHash, destPath, srcDirEnt)
	if err != nil {
		common.Die("error inserting src into folder: %s\n", err.Error())
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}

	ok, err = remote.CasTag(c, *tagArg, hex.EncodeToString(destHash[:]), hex.EncodeToString(newRoot.Data[:]), generation)
	if err != nil {
		common.Die("creating tag: %s\n", err.Error())
	}

	if !ok {
		// XXX: loop here
		common.Die("tag concurrently modified, try again\n")
	}

}
