package p9

import (
	"flag"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/cmd/bpy/p9/proto9"
	"github.com/buppyio/bpy/remote"
	"log"
	"net"
)

func handleConnection(con net.Conn, ref string) {
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

	refHash, ok, err := remote.GetRef(c, ref)
	if err != nil {
		log.Fatalf("error fetching ref hash: %s\n", err.Error())
	}

	if !ok {
		log.Fatalf("ref '%s' does not exist\n", ref)
	}

	root, err := bpy.ParseHash(refHash)
	if err != nil {
		common.Die("error parsing hash: %s\n", err.Error())
	}

	maxMessageSize := uint32(1024 * 1024)
	srv := &Server{
		rwc:            con,
		maxMessageSize: maxMessageSize,
		fids:           make(map[proto9.Fid]Handle),
		client:         c,
		store:          store,
		root:           root,
	}
	srv.Serve()

	defer store.Close()
}

func P9() {
	refArg := flag.String("ref", "default", "ref of directory to list")
	addrArg := flag.String("addr", "127.0.0.1:9001", "address to listen on ")
	flag.Parse()

	if *refArg == "" {
		log.Fatalf("please specify a ref to browse\n")
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
		go handleConnection(con, *refArg)
	}
}
