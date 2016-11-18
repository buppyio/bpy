package listpacks

import (
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/remote"
)

func ListPacks() {
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

	packs, err := remote.ListPacks(c)
	if err != nil {
		common.Die("error listing packs: %s\n", err.Error())
	}
	for _, pack := range packs {
		_, err := fmt.Printf("%s %d\n", pack.Name, pack.Size)
		if err != nil {
			common.Die("error pritning pack: %s\n", err.Error())
		}
	}
}
