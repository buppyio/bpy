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
	flag.Parse()

	cfg, err := common.GetConfig()
	if err != nil {
		common.Die("error getting config: %s\n", err)
	}

	k, err := common.GetKey(cfg)
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(cfg, &k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	store, err := common.GetCStore(cfg, &k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	rootHash, _, ok, err := remote.GetRoot(c, &k)
	if err != nil {
		common.Die("error fetching root hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("root missing\n")
	}

	for {
		ref, err := refs.GetRef(store, rootHash)
		if err != nil {
			common.Die("error fetching ref: %s\n", err.Error())
		}
		_, err = fmt.Printf("%s@%s\n", hex.EncodeToString(rootHash[:]), time.Unix(ref.CreatedAt, 0))
		if err != nil {
			common.Die("io error: %s\n", err.Error())
		}
		if !ref.HasPrev {
			break
		}
		rootHash = ref.Prev
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing content store: %s\n", err.Error())
	}
}
