package browse

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/remote"
	"acha.ninja/bpy/remote/client"
	"errors"
	"flag"
	"fmt"
	"github.com/pkg/browser"
	"log"
	"net/http"
	"os"
	"time"
)

type httpFs struct {
	c     *client.Client
	store bpy.CStore
	tag   string
}

func (httpFs *httpFs) Open(path string) (http.File, error) {
	log.Printf("open: %s", path)
	tag, ok, err := remote.GetTag(httpFs.c, httpFs.tag)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("tag '%s' does not exist", httpFs.tag)
	}
	root, err := bpy.ParseHash(tag)
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
	tagArg := flag.String("tag", "default", "tag of directory to list")
	addrArg := flag.String("addr", "127.0.0.1:8080", "address to listen on ")
	flag.Parse()

	if *tagArg == "" {
		common.Die("please specify a tag to browse\n")
	}

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

	url := "http://" + *addrArg
	log.Printf("serving on %s\n", url)
	go func() {
		time.Sleep(100 * time.Millisecond)
		browser.OpenURL(url)
	}()
	log.Fatal(http.ListenAndServe(*addrArg, http.FileServer(&httpFs{
		c:     c,
		store: cstore.NewMemCachedCStore(store, 64*1024*1024),
		tag:   *tagArg,
	})))
}
