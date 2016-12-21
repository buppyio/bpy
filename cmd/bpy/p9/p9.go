package p9

import (
	"flag"
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/cmd/bpy/p9/server9"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
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

	root, _, ok, err := remote.GetRoot(c, &k)
	if err != nil {
		common.Die("error getting root: %s", err)
	}
	if !ok {
		common.Die("root missing\n")
	}

	ref, err := refs.GetRef(store, root)
	if err != nil {
		common.Die("error getting ref: %s", err)
	}

	dirEnts, err := fs.ReadDir(store, ref.Root)
	if err != nil {
		common.Die("error reading root: %s", err)
	}

	attachFunc := func(name string) (server9.File, error) {

		fs := &fs9{
			key:    k,
			store:  store,
			client: c,
		}

		f, err := fs.CreateFile(dirEnts[0], nil, "/")
		if err != nil {
			return nil, fmt.Errorf("error creating root file: %s", err)
		}
		fs.file = f

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
