package ref

import (
	"flag"
	"fmt"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/remote"
	"os"
)

func refhelp() {
	fmt.Println("Please specify one of the following subcommands:")
	fmt.Println("create, get, list, remove, cas")
	os.Exit(1)
}

func create() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		common.Die("please specity a ref and a hash\n")
	}
	_, err := bpy.ParseHash(flag.Args()[1])
	if err != nil {
		common.Die("hash not valid: %s\n", err.Error())
	}
	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}
	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error getting remote: %s\n", err.Error())
	}
	defer c.Close()

	generation, err := remote.GetGeneration(c)
	if err != nil {
		common.Die("error getting current gc generation: %s\n", err.Error())
	}

	err = remote.NewRef(c, flag.Args()[0], flag.Args()[1], generation)
	if err != nil {
		common.Die("error create ref: %s\n", err.Error())
	}
}

func get() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		common.Die("please specity a ref\n")
	}
	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}
	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error getting remote: %s\n", err.Error())
	}
	defer c.Close()

	hash, ok, err := remote.GetRef(c, flag.Args()[0])
	if err != nil {
		common.Die("error setting ref: %s\n", err.Error())
	}
	if !ok {
		common.Die("ref '%s' does not exist", flag.Args()[0])
	}
	_, err = fmt.Println(hash)
	if err != nil {
		common.Die("io error: %s\n", err.Error())
	}
}

func remove() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		common.Die("please specity a ref and its value\n")
	}
	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}
	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error getting remote: %s\n", err.Error())
	}
	defer c.Close()

	generation, err := remote.GetGeneration(c)
	if err != nil {
		common.Die("error getting current gc generation: %s\n", err.Error())
	}

	err = remote.RemoveRef(c, flag.Args()[0], flag.Args()[1], generation)
	if err != nil {
		common.Die("error removing ref: %s\n", err.Error())
	}
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
	reflist, err := remote.ListRefs(c)
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
		case "create":
			cmd = create
		case "get":
			cmd = get
		case "remove":
			cmd = remove
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
