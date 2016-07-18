package remote

import (
	"acha.ninja/bpy/remote/server"
	"flag"
	"io"
	"log"
	"os"
)

type remoteInputOutput struct {
	stdin  io.ReadCloser
	stdout io.WriteCloser
}

func (p *remoteInputOutput) Read(buf []byte) (int, error)  { return p.stdin.Read(buf) }
func (p *remoteInputOutput) Write(buf []byte) (int, error) { return p.stdout.Write(buf) }
func (p *remoteInputOutput) Close() error                  { p.stdin.Close(); p.stdout.Close(); return nil }

func Remote() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatal("please specify a directory\n")
	}
	server.Serve(&remoteInputOutput{
		stdin:  os.Stdin,
		stdout: os.Stdout,
	}, flag.Args()[0])
}
