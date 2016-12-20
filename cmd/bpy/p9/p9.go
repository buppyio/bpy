package p9

import (
	"flag"
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/cmd/bpy/p9/server9"
	"log"
	"net"
)

func handleConnection(con net.Conn, tag string) {
	defer con.Close()
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
	defer store.Close()

	attachFunc := func(name string) (server9.File, error) {
		return nil, fmt.Errorf("unimplemented")
	}
	srv := server9.NewServer(1024*1024, attachFunc)
	srv.Serve(con)
}

func P9() {
	tagArg := flag.String("tag", "default", "tag of directory to list")
	addrArg := flag.String("addr", "127.0.0.1:9001", "address to listen on ")
	flag.Parse()

	if *tagArg == "" {
		log.Fatalf("please specify a tag to browse\n")
	}

	log.Printf("listening on: %s", *addrArg)
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
