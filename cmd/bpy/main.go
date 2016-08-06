package main

import (
	"acha.ninja/bpy/cmd/bpy/browse"
	"acha.ninja/bpy/cmd/bpy/cat"
	"acha.ninja/bpy/cmd/bpy/cp"
	"acha.ninja/bpy/cmd/bpy/get"
	"acha.ninja/bpy/cmd/bpy/ls"
	"acha.ninja/bpy/cmd/bpy/mkdir"
	"acha.ninja/bpy/cmd/bpy/mv"
	"acha.ninja/bpy/cmd/bpy/newkey"
	"acha.ninja/bpy/cmd/bpy/p9"
	"acha.ninja/bpy/cmd/bpy/put"
	"acha.ninja/bpy/cmd/bpy/remote"
	"acha.ninja/bpy/cmd/bpy/rm"
	"acha.ninja/bpy/cmd/bpy/tag"
	"acha.ninja/bpy/cmd/bpy/tar"
	"acha.ninja/bpy/cmd/bpy/zip"
	"fmt"
	"os"
)

func help() {
	fmt.Println("Please specify one of the following subcommands:")
	fmt.Println("browse, cat, cp, get, ls, mkdir, mv, new-key, put, rm, tag, tar, zip, 9p")
	fmt.Println("")
	fmt.Println("For more use -h on the sub commands.")
	fmt.Println("Also check the docs at https://buppy.io/docs")
	os.Exit(1)
}

func main() {
	cmd := help
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "browse":
			cmd = browse.Browse
		case "cat":
			cmd = cat.Cat
		case "cp":
			cmd = cp.Cp
		case "get":
			cmd = get.Get
		case "ls":
			cmd = ls.Ls
		case "mkdir":
			cmd = mkdir.Mkdir
		case "mv":
			cmd = mv.Mv
		case "put":
			cmd = put.Put
		case "remote":
			cmd = remote.Remote
		case "rm":
			cmd = rm.Rm
		case "tag":
			cmd = tag.Tag
		case "tar":
			cmd = tar.Tar
		case "dbg":
			cmd = dbg
		case "new-key":
			cmd = newkey.NewKey
		case "zip":
			cmd = zip.Zip
		case "9p":
			cmd = p9.P9
		default:
			cmd = help
		}
		copy(os.Args[1:], os.Args[2:])
		os.Args = os.Args[0 : len(os.Args)-1]
	}
	cmd()
}
