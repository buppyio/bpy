package p9

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/cmd/bpy/p9/proto9"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"log"
	"net"
)

func handleConnection(con net.Conn) {
	defer con.Close()
	k, err := common.GetKey()
	if err != nil {
		log.Fatalf("error getting key: %s\n", err.Error())
	}

	c, err := common.GetRemote(&k)
	if err != nil {
		log.Fatalf("error connecting to remote: %s", err.Error())
	}
	defer c.Close()

	store, err := common.GetCStore(&k, c)
	if err != nil {
		log.Printf("error getting content store: %s", err.Error())
		return
	}

	_, rootHash, ok, err := remote.GetRoot(c, &k)
	if err != nil {
		log.Fatalf("error fetching root hash: %s", err.Error())
	}
	if !ok {
		log.Fatalf("root missing")
	}

	ref, err := refs.GetRef(store, rootHash)
	if err != nil {
		common.Die("error fetching ref: %s", err.Error())
	}

	maxMessageSize := uint32(1024 * 1024)
	srv := &Server{
		rwc:            con,
		maxMessageSize: maxMessageSize,
		fids:           make(map[proto9.Fid]Handle),
		client:         c,
		store:          store,
		root:           ref.Root,
	}
	srv.Serve()

	defer store.Close()
}

func P9() {
	addrArg := flag.String("addr", "127.0.0.1:9001", "address to listen on ")
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
