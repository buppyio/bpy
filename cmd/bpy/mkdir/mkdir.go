package mkdir

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
)

func Mkdir() {
	refArg := flag.String("ref", "default", "ref make directory in")
	flag.Parse()

	if len(flag.Args()) != 1 {
		common.Die("please specify the directory to create\n")
	}

	destPath := flag.Args()[0]

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

	ref, ok, err := remote.GetRef(c, &k, *refArg)
	if err != nil {
		common.Die("error fetching ref hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("ref '%s' does not exist\n", *refArg)
	}

	emptyDirEnt, err := fs.EmptyDir(store, 0755)
	if err != nil {
		common.Die("error copying data: %s\n", err.Error())
	}

	newRootEnt, err := fs.Insert(store, ref.Root, destPath, emptyDirEnt)
	if err != nil {
		common.Die("error inserting empty dir into folder: %s\n", err.Error())
	}
	newRef := refs.Ref{
		Root: newRootEnt.HTree.Data,
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
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
