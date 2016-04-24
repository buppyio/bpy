package main

import (
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/proto9"
	"net"
	"os"
)

func main() {
	con, err := net.Dial("tcp", os.Args[1])
	if err != nil {
		panic(err)
	}
	defer con.Close()
	c, err := client9.NewClient(proto9.NewConn(con, con, 65536))
	if err != nil {
		panic(err)
	}
	err = c.Attach("ac", "")
	if err != nil {
		panic(err)
	}
	if err != nil {
		panic(err)
	}
	f, err := c.Open(os.Args[2], proto9.OREAD)
	if err != nil {
		panic(err)
	}
	err = f.Close()
	if err != nil {
		panic(err)
	}
}
