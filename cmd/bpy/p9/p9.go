package p9

import (
	"acha.ninja/bpy/cmd/bpy/common"
	"flag"
	"log"
	"net"
)

func handleConnection(con net.Conn, tag string) {
	defer con.Close()
	k, err := common.GetKey()
	if err != nil {
		log.Fatalf("error getting key: %s\n", err.Error())
	}

	c, err := common.GetRemote(&k)
	if err != nil {
		log.Fatalf("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	store, err := common.GetCStore(&k, c)
	if err != nil {
		log.Printf("error getting content store: %s\n", err.Error())
		return
	}
	defer store.Close()
}

func P9() {
	tagArg := flag.String("tag", "default", "tag of directory to list")
	addrArg := flag.String("addr", "127.0.0.1:8080", "address to listen on ")
	flag.Parse()

	if *tagArg == "" {
		log.Fatalf("please specify a tag to browse\n")
	}

	listener, err := net.Listen("tcp", *addrArg)
	if err != nil {
		log.Fatalf("error listening: %s", err.Error())
	}
	for {
		con, err := listener.Accept()
		if err != nil {
			log.Fatalf("error accepting connection: %s", err.Error())
		}
		go handleConnection(con, *tagArg)
	}
}
