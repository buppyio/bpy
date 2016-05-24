package main

import (
	"acha.ninja/bpy/cmd/bpy/cat"
	"acha.ninja/bpy/cmd/bpy/get"
	"acha.ninja/bpy/cmd/bpy/ls"
	"acha.ninja/bpy/cmd/bpy/put"
	"acha.ninja/bpy/cmd/bpy/remote"
	"acha.ninja/bpy/cmd/bpy/tag"
	"fmt"
	"os"
)

func help() {
	fmt.Println("Please specify one of the following subcommands:")
	fmt.Println("put, get, cat, ls, tag, help\n")
	fmt.Println("For more use -h on the sub commands.")
	fmt.Println("Also check the docs at https://buppy.io/docs")
	os.Exit(1)
}

func main() {
	cmd := help
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "put":
			cmd = put.Put
		case "get":
			cmd = get.Get
		case "cat":
			cmd = cat.Cat
		case "ls":
			cmd = ls.Ls
		case "remote":
			cmd = remote.Remote
		case "tag":
			cmd = tag.Tag
		case "dbg":
			cmd = dbg
		default:
			cmd = help
		}
		copy(os.Args[1:], os.Args[2:])
		os.Args = os.Args[0 : len(os.Args)-1]
	}
	cmd()
}
