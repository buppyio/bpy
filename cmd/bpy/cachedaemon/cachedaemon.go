package cachedaemon

import (
	"flag"
	"github.com/buppyio/bpy/cstore/cache"
	"log"
	"net"
)

func CacheDaemon() {
	dbArg := flag.String("db", "", "path to dbfile")
	addrArg := flag.String("addr", "127.0.0.1:9001", "address to listen on")
	sizeArg := flag.Int64("size", 1024*1024*1024, "max size of cache in bytes")
	flag.Parse()

	if *dbArg == "" {
		log.Fatalf("please specify a db file")
	}

	server, err := cache.NewServer(*dbArg, 0755, uint64(*sizeArg))
	if err != nil {
		log.Fatalf("error creating cache server: %s", err.Error())
	}

	log.Printf("listening on %s", *addrArg)
	l, err := net.Listen("tcp", *addrArg)
	if err != nil {
		log.Fatalf("error listening: %s", err.Error())
	}
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatalf("error Accepting: %s", err.Error())
		}
		go server.ServeConn(c)
	}

}
