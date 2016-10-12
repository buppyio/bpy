package mkdir

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"time"
)

func Mkdir() {
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

	generation, err := remote.GetGeneration(c)
	if err != nil {
		common.Die("error getting current gc generation: %s\n", err.Error())
	}

	store, err := common.GetCStore(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	rootVersion, rootHash, ok, err := remote.GetRoot(c, &k)
	if err != nil {
		common.Die("error fetching root hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("root missing\n")
	}

	ref, err := refs.GetRef(store, rootHash)
	if err != nil {
		common.Die("error fetching ref: %s\n", err.Error())
	}

	emptyDirEnt, err := fs.EmptyDir(store, 0755)
	if err != nil {
		common.Die("error copying data: %s\n", err.Error())
	}

	newRootEnt, err := fs.Insert(store, ref.Root, destPath, emptyDirEnt)
	if err != nil {
		common.Die("error inserting empty dir into folder: %s\n", err.Error())
	}

	newRefHash, err := refs.PutRef(store, refs.Ref{
		CreatedAt: time.Now().Unix(),
		Root:      newRootEnt.HTree.Data,
		HasPrev:   true,
		Prev:      rootHash,
	})

	err = store.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}

	ok, err = remote.CasRoot(c, &k, rootVersion+1, newRefHash, generation)
	if err != nil {
		common.Die("swapping root: %s\n", err.Error())
	}

	if !ok {
		// XXX: loop here
		common.Die("root concurrently modified, try again\n")
	}

}
