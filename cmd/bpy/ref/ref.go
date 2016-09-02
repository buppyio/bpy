package ref

import (
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/remote"
	"os"
)

func refhelp() {
	fmt.Println("Please specify one of the following subcommands:")
	fmt.Println("list")
	os.Exit(1)
}

func list() {
	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}
	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error getting remote: %s", err.Error())
	}
	defer c.Close()
	reflist, err := remote.ListNamedRefs(c)
	if err != nil {
		common.Die("error getting ref list: %s", err.Error())
	}
	for _, t := range reflist {
		_, err := fmt.Println(t)
		if err != nil {
			common.Die("io error: %s\n", err.Error())
		}
	}
}

func Ref() {
	cmd := refhelp
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			cmd = list
		default:
			cmd = refhelp
		}
		copy(os.Args[1:], os.Args[2:])
		os.Args = os.Args[0 : len(os.Args)-1]
	}
	cmd()
}
