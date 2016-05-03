package remote

import (
	"acha.ninja/bpy/cstore/export"
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
	export, err := export.NewExportServer(&remoteInputOutput{
		stdin:  os.Stdin,
		stdout: os.Stdout,
	}, os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(export.Serve())
}
