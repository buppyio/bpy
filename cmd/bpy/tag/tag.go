package tag

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/tags"
	"flag"
	"fmt"
	"os"
)

func taghelp() {
	fmt.Println("Please specify one of the following subcommands:")
	fmt.Println("create, get, list, remove, cas")
	os.Exit(1)
}

func create() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		common.Die("please specity a tag and a hash\n")
	}
	_, err := bpy.ParseHash(flag.Args()[1])
	if err != nil {
		common.Die("hash not valid: %s\n", err.Error())
	}
	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}
	remote, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error getting remote: %s\n", err.Error())
	}
	err = tags.Create(remote, flag.Args()[0], flag.Args()[1])
	if err != nil {
		common.Die("error create tag: %s\n", err.Error())
	}
}

func get() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		common.Die("please specity a tag\n")
	}
	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}
	remote, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error getting remote: %s\n", err.Error())
	}
	hash, err := tags.Get(remote, flag.Args()[0])
	if err != nil {
		common.Die("error setting tag: %s\n", err.Error())
	}
	_, err = fmt.Println(hash)
	if err != nil {
		common.Die("io error: %s\n", err.Error())
	}
}

func remove() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		common.Die("please specity a tag and its value\n")
	}
	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}
	remote, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error getting remote: %s\n", err.Error())
	}
	err = tags.Remove(remote, flag.Args()[0], flag.Args()[1])
	if err != nil {
		common.Die("error removing tag: %s\n", err.Error())
	}
}

func cas() {
	flag.Parse()
	if len(flag.Args()) != 3 {
		common.Die("please specity a tag, the old hash and the new hash\n")
	}
	_, err := bpy.ParseHash(flag.Args()[1])
	if err != nil {
		common.Die("old hash not valid: %s\n", err.Error())
	}
	_, err = bpy.ParseHash(flag.Args()[2])
	if err != nil {
		common.Die("new hash not valid: %s\n", err.Error())
	}
	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}
	remote, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error getting remote: %s\n", err.Error())
	}
	err = tags.Cas(remote, flag.Args()[0], flag.Args()[1], flag.Args()[2])
	if err != nil {
		common.Die("error setting tag: %s\n", err.Error())
	}
}

func list() {
	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}
	remote, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error getting remote: %s", err.Error())
	}
	taglist, err := tags.List(remote)
	if err != nil {
		common.Die("error getting tag list: %s", err.Error())
	}
	for _, t := range taglist {
		_, err := fmt.Println(t)
		if err != nil {
			common.Die("io error: %s\n", err.Error())
		}
	}
}

func Tag() {
	cmd := taghelp
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "create":
			cmd = create
		case "cas":
			cmd = cas
		case "get":
			cmd = get
		case "remove":
			cmd = remove
		case "list":
			cmd = list
		default:
			cmd = taghelp
		}
		copy(os.Args[1:], os.Args[2:])
		os.Args = os.Args[0 : len(os.Args)-1]
	}
	cmd()
}
