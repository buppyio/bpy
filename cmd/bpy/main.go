package main

import (
	"acha.ninja/bpy/cmd/bpy/cat"
	"acha.ninja/bpy/cmd/bpy/get"
	"acha.ninja/bpy/cmd/bpy/ls"
	"acha.ninja/bpy/cmd/bpy/put"
	"acha.ninja/bpy/cmd/bpy/remote"
	"acha.ninja/bpy/cmd/bpy/srv9p"
	"os"
)

func main() {
	switch os.Args[1] {
	case "put":
		put.Put()
	case "get":
		get.Get()
	case "cat":
		cat.Cat()
	case "ls":
		ls.Ls()
	case "9p":
		srv9p.Srv9p()
	case "remote":
		remote.Remote()
	case "dbg":
		dbg()
	}
}
