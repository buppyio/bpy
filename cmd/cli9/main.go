package main

import (
	"acha.ninja/bpy/client9"
	"net"
	"os"
)

func dial(addr string) (*client9.Client, error) {
	con, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	c, err := client9.NewClient(proto9.NewConn(con, con, 65536))
	if err != nil {
		return nil, err
	}
	err = c.Attach()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func main() {
	c, err := dial(os.Args[1])
	if err != nil {
		panic(err)
	}
	
	f, err := c.Open(os.Args[2])
	if err != nil {
		panic(err)
	}
	err = f.Close()
	if err != nil {
		panic(err)
	}
}
