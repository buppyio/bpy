package main

import (
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/browse"
	"github.com/buppyio/bpy/cmd/bpy/cat"
	"github.com/buppyio/bpy/cmd/bpy/cp"
	"github.com/buppyio/bpy/cmd/bpy/gc"
	"github.com/buppyio/bpy/cmd/bpy/get"
	"github.com/buppyio/bpy/cmd/bpy/ls"
	"github.com/buppyio/bpy/cmd/bpy/mkdir"
	"github.com/buppyio/bpy/cmd/bpy/mv"
	"github.com/buppyio/bpy/cmd/bpy/newkey"
	"github.com/buppyio/bpy/cmd/bpy/p9"
	"github.com/buppyio/bpy/cmd/bpy/put"
	"github.com/buppyio/bpy/cmd/bpy/remote"
	"github.com/buppyio/bpy/cmd/bpy/rm"
	"github.com/buppyio/bpy/cmd/bpy/tag"
	"github.com/buppyio/bpy/cmd/bpy/tar"
	"github.com/buppyio/bpy/cmd/bpy/zip"
	"os"
)

func help() {
	fmt.Println("Please specify one of the following subcommands:")
	fmt.Println("browse, cat, cp, gc, get, ls, mkdir, mv, new-key, put, rm, tag, tar, zip, 9p")
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
		case "gc":
			cmd = gc.GC
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
