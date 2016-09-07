package hist

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"time"
)

func Hist() {
	refArg := flag.String("ref", "default", "ref to fetch history for")

	flag.Parse()

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

	refHash, ok, err := remote.GetNamedRef(c, &k, *refArg)
	if err != nil {
		common.Die("error fetching ref hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("ref '%s' does not exist\n", *refArg)
	}

	for {
		ref, err := refs.GetRef(store, refHash)
		if err != nil {
			common.Die("error fetching ref: %s\n", err.Error())
		}
		_, err = fmt.Printf("%s@%s\n", hex.EncodeToString(refHash[:]), time.Unix(ref.CreatedAt, 0))
		if err != nil {
			common.Die("io error: %s\n", err.Error())
		}
		if !ref.HasPrev {
			break
		}
		refHash = ref.Prev
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing content store: %s\n", err.Error())
	}
}
