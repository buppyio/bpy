package browse

/*
import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/tags"
	"flag"
)
*/

func Browse() {
	/*
		var root [32]byte
		tagArg := flag.String("tag", "", "tag of directory to list")
		hashArg := flag.String("hash", "", "hash of directory to list")
		flag.Parse()

		if *hashArg == "" && *tagArg == "" || *hashArg != "" && *tagArg != "" {
			common.Die("please specify a hash or a tag to list\n")
		}

		if *hashArg != "" {
			hash, err := bpy.ParseHash(*hashArg)
			if err != nil {
				common.Die("error parsing hash: %s\n", err.Error())
			}
			root = hash
		}

		if len(flag.Args()) != 2 {
			common.Die("please specify a src path and a dest path\n")
		}

		remote, err := common.GetRemote()
		if err != nil {
			common.Die("error connecting to remote: %s\n", err.Error())
		}

		store, err := common.GetCStoreReader(remote)
		if err != nil {
			common.Die("error getting content store: %s\n", err.Error())
		}

		if *tagArg != "" {
			tagHash, err := tags.Get(remote, *tagArg)
			if err != nil {
				common.Die("error fetching tag hash: %s\n", err.Error())
			}
			root, err = bpy.ParseHash(tagHash)
			if err != nil {
				common.Die("error parsing hash: %s\n", err.Error())
			}
		}

		err = store.Close()
		if err != nil {
			common.Die("error closing remote: %s\n", err.Error())
		}
	*/
}
