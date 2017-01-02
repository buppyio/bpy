package cachedaemon

import (
	"flag"
	"github.com/buppyio/bpy/cstore/cache"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func deadman(newCon, conOk, conClosed, shutdown chan struct{}, idleTimeout time.Duration) {
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
	shutdown <- struct{}{}
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

func listenLoop(newCon, conOk, conClosed chan struct{}, l net.Listener, server *rpc.Server) {
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

func CacheDaemon() {
	dbArg := flag.String("db", "", "path to dbfile")
	socketTypeArg := flag.String("socktype", "", "type of socket to listen on")
	addrArg := flag.String("addr", "", "address to listen on")
	nohupArg := flag.Bool("nohup", false, "ignore HUP signals")
	sizeArg := flag.Int64("size", 1024*1024*1024, "max size of cache in bytes")
	idleTimeoutArg := flag.Int64("idle-timeout", -1, "close if no connections after this many seconds")
	flag.Parse()

	// Set a umask so unix sockets are not public
	syscall.Umask(0077)

	if *nohupArg {
		signal.Ignore(syscall.SIGHUP)
	}

	if *dbArg == "" {
		log.Fatalf("please specify a db file")
	}

	c, err := cache.NewCache(*dbArg, 0600, uint64(*sizeArg))
	if err != nil {
		log.Fatalf("error creating: %s", err.Error())
	}

	server, err := cache.NewServer(c)
	if err != nil {
		log.Fatalf("error creating cache server: %s", err.Error())
	}

	// The boltdb locking means we are the only process with the cache db open.
	if *socketTypeArg == "unix" {
		os.Remove(*addrArg)
	}

	log.Printf("listening on %s %s", *socketTypeArg, *addrArg)
	l, err := net.Listen(*socketTypeArg, *addrArg)
	if err != nil {
		log.Fatalf("error listening: %s", err.Error())
	}

	newCon := make(chan struct{})
	conOk := make(chan struct{})
	conClosed := make(chan struct{})
	shutdown := make(chan struct{})

	if *idleTimeoutArg < 0 {
		go runForever(newCon, conOk, conClosed)
	} else {
		go deadman(newCon, conOk, conClosed, shutdown, time.Second*time.Duration(*idleTimeoutArg))
	}

	go listenLoop(newCon, conOk, conClosed, l, server)

	<-shutdown
	log.Printf("exiting due expired idle timer")
	c.Close()
	os.Exit(0)

}
