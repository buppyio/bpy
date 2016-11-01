package cachedaemon

import (
	"flag"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/cstore/cache"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func deadman(newCon, conOk, conClosed chan struct{}, idleTimeout time.Duration) {
	t := time.NewTimer(idleTimeout)
	counter := uint64(0)

	for {
		select {
		case <-newCon:
			counter += 1
			t.Stop()
			select {
			case <-t.C:
				goto timeout
			default:
			}
			conOk <- struct{}{}
		case <-conClosed:
			counter -= 1
			if counter == 0 {
				if t.Reset(idleTimeout) {
					goto timeout
				}
			}
		case <-t.C:
			goto timeout
		}
	}

timeout:

	log.Printf("exiting due expired idle timer")
	os.Exit(0)
}

func runForever(newCon, conOk, conClosed chan struct{}) {
	for {
		select {
		case <-newCon:
			conOk <- struct{}{}
		case <-conClosed:
		}
	}

}

func CacheDaemon() {
	dbArg := flag.String("db", "", "path to dbfile")
	addrArg := flag.String("addr", common.DefaultCacheListenAddr, "address to listen on")
	nohupArg := flag.Bool("nohup", false, "ignore HUP signals")
	sizeArg := flag.Int64("size", 1024*1024*1024, "max size of cache in bytes")
	idleTimeoutArg := flag.Int64("idle-timeout", -1, "close if no connections after this many seconds")
	flag.Parse()

	if *nohupArg {
		signal.Ignore(syscall.SIGHUP)
	}

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

	newCon := make(chan struct{})
	conOk := make(chan struct{})
	conClosed := make(chan struct{})

	if *idleTimeoutArg < 0 {
		go runForever(newCon, conOk, conClosed)
	} else {
		go deadman(newCon, conOk, conClosed, time.Second*time.Duration(*idleTimeoutArg))
	}

	for {
		c, err := l.Accept()

		if err != nil {
			log.Fatalf("error Accepting: %s", err.Error())
		}

		log.Printf("new connection from %s", c.RemoteAddr())
		newCon <- struct{}{}
		select {
		case <-conOk:
			go func() {
				server.ServeConn(c)
				log.Printf("%s disconnected", c.RemoteAddr())
				c.Close()
				conClosed <- struct{}{}
			}()
		}
	}

}
