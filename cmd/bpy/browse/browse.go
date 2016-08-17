package browse

import (
	"errors"
	"flag"
	"fmt"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/remote/client"
	"github.com/pkg/browser"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var head = []byte(`<!DOCTYPE html>
<html lang="en">
<head>
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
<meta http-equiv="Content-Language" content="en" />
<title>bpy - file browser</title>
<style>
body {
	font-family: monospace;
	color: #000;
	background-color: #fff;
}

table thead td {
	font-weight: bold;
}

table td {
	padding: 0 0.4em;
}

#content table td {
	white-space: nowrap;
	vertical-align: top;
}

#files tr:hover td {
	background-color: #eee;
}

</style>
</head>
<body>
`)

var tail = []byte(`</body>
</html>
`)

type rootHandler struct {
	c     *client.Client
	store bpy.CStore
}

func (h *rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write(head)
	w.Write(tail)
}

type refHandler struct {
	c     *client.Client
	store bpy.CStore
}

func (h *refHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write(head)
	w.Write(tail)
}

type httpFs struct {
	c     *client.Client
	store bpy.CStore
}

func (httpFs *httpFs) Open(fullPath string) (http.File, error) {
	log.Printf("open: %s", fullPath)
	idx := strings.Index(fullPath, "/")
	if idx == -1 {
		return nil, fmt.Errorf("invalid path %s", fullPath)
	}
	refName, path := fullPath[:idx], fullPath[idx:]
	ref, ok, err := remote.GetRef(httpFs.c, refName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("ref '%s' does not exist", refName)
	}
	root, err := bpy.ParseHash(ref)
	if err != nil {
		return nil, err
	}
	ent, err := fs.Walk(httpFs.store, root, path)
	if err != nil {
		return nil, err
	}

	if ent.EntMode.IsRegular() {
		rdr, err := fs.Open(httpFs.store, root, path)
		if err != nil {
			return nil, err
		}
		return &httpFile{
			httpFs: httpFs,
			ent:    ent,
			rdr:    rdr,
		}, nil
	}

	if ent.EntMode.IsDir() {
		return &httpDir{
			httpFs: httpFs,
			root:   root,
			path:   path,
			ent:    ent,
		}, nil
	}

	return nil, errors.New("cannot serve this file type")
}

type httpFile struct {
	httpFs *httpFs
	ent    fs.DirEnt
	rdr    *fs.FileReader
}

func (f *httpFile) Seek(offset int64, whence int) (int64, error) {
	return f.rdr.Seek(offset, whence)
}

func (f *httpFile) Read(buf []byte) (int, error) {
	return f.rdr.Read(buf)
}

func (f *httpFile) Close() error {
	return f.rdr.Close()
}

func (f *httpFile) Stat() (os.FileInfo, error) {
	return &f.ent, nil
}

func (f *httpFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("not a dir")
}

type httpDir struct {
	httpFs *httpFs
	root   [32]byte
	path   string
	ent    fs.DirEnt
}

func (d *httpDir) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("not a file")
}

func (d *httpDir) Read(buf []byte) (int, error) {
	return 0, errors.New("not a file")
}

func (d *httpDir) Close() error {
	return errors.New("not a file")
}

func (d *httpDir) Stat() (os.FileInfo, error) {
	return &d.ent, nil
}

func (d *httpDir) Readdir(count int) ([]os.FileInfo, error) {
	ents, err := fs.Ls(d.httpFs.store, d.root, d.path)
	if err != nil {
		return nil, err
	}
	ents = ents[1:]
	finfo := make([]os.FileInfo, len(ents), len(ents))
	for idx := range ents {
		finfo[idx] = &ents[idx]
	}
	if count >= 0 && count < len(finfo) {
		finfo = finfo[:count]
	}
	return finfo, nil
}

func Browse() {
	addrArg := flag.String("addr", "127.0.0.1:8080", "address to listen on ")
	flag.Parse()

	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting key: %s\n", err.Error())
	}

	log.Printf("connecting to remote\n")
	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}

	store, err := common.GetCStore(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	http.Handle("/", &rootHandler{
		c: c,
	})

	http.Handle("/refs/", &refHandler{
		c:     c,
		store: store,
	})

	http.Handle("/raw/", http.StripPrefix("/raw/", http.FileServer(&httpFs{
		c:     c,
		store: store,
	})))

	url := "http://" + *addrArg
	log.Printf("serving on %s\n", url)

	go func() {
		time.Sleep(100 * time.Millisecond)
		browser.OpenURL(url)
	}()

	log.Fatal(http.ListenAndServe(*addrArg, nil))
}
