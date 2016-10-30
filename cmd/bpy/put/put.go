package put

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/fs/fsutil"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"path/filepath"
	"time"
)

func Put() {
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

	for {
		store, err := common.GetCStore(&k, c)
		if err != nil {
			common.Die("error getting content store: %s\n", err.Error())
		}

		rootHash, rootVersion, ok, err := remote.GetRoot(c, &k)
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

		srcDirEnt, err := fsutil.CpHostToFs(store, srcPath)
		if err != nil {
			common.Die("error copying data: %s\n", err.Error())
		}

		newRootEnt, err := fs.Insert(store, ref.Root, destPath, srcDirEnt)
		if err != nil {
			common.Die("error inserting src into folder: %s\n", err.Error())
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

		ok, err = remote.CasRoot(c, &k, newRefHash, rootVersion+1, generation)
		if err != nil {
			common.Die("error swapping root: %s\n", err.Error())
		}
		if ok {
			break
		}
	}
}
