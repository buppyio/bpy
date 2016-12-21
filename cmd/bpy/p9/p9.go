package p9

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/cmd/bpy/p9/server9"
	"log"
	"net"
)

func handleConnection(con net.Conn) {
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

		fs := &fs9{
			key:    k,
			store:  store,
			client: c,
		}

		return fs, nil
	}
	srv := server9.NewServer(1024*1024, attachFunc)
	srv.Serve(con)
}

func P9() {
	addrArg := flag.String("addr", "127.0.0.1:9001", "address to listen on")
	flag.Parse()

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
		go handleConnection(con)
	}
}
