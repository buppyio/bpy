package browse

import (
	"errors"
	"flag"
	"fmt"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/archive"
	"github.com/buppyio/bpy/cmd/bpy/browse/static"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/remote/client"
	"github.com/pkg/browser"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"time"
)

type rootHandler struct {
	c     *client.Client
	k     *bpy.Key
	store bpy.CStore
}

func (h *rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	walkPath := r.URL.Path[1:]
	rootHash, _, ok, err := remote.GetRoot(h.c, h.k)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error: %s", err.Error())
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "root missing\n")
		return
	}

	ref, err := refs.GetRef(h.store, rootHash)
	if err != nil {
		common.Die("error fetching ref: %s\n", err.Error())
	}

	ent, err := fs.Walk(h.store, ref.Root, walkPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error: %s", err.Error())
		return
	}

	dirEnts, err := fs.ReadDir(h.store, ent.HTree.Data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error: %s", err.Error())
		return
	}

	_ = browseTemplate.Execute(w, struct {
		Path    string
		DirEnts fs.DirEnts
	}{
		Path:    walkPath,
		DirEnts: dirEnts,
	})
}

type zipHandler struct {
	c     *client.Client
	k     *bpy.Key
	store bpy.CStore
}

func (h *zipHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	walkPath := r.URL.Path[1:]
	rootHash, _, ok, err := remote.GetRoot(h.c, h.k)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error: %s", err.Error())
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "root missing\n")
		return
	}

	ref, err := refs.GetRef(h.store, rootHash)
	if err != nil {
		common.Die("error fetching ref: %s\n", err.Error())
	}

	ent, err := fs.Walk(h.store, ref.Root, walkPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error: %s", err.Error())
		return
	}

	if !ent.IsDir() {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: not a directory")
		return
	}

	base := path.Base(walkPath)
	zipName := ""
	switch base {
	case ".":
		fallthrough
	case "/":
		zipName = "root.zip"
	default:
		zipName = base + ".zip"
	}
	log.Printf("%s", zipName)

	w.Header().Set("Content-Type", mime.TypeByExtension(".zip"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipName))

	archive.Zip(h.store, ent.HTree.Data, w)
}

type httpFs struct {
	c     *client.Client
	k     *bpy.Key
	store bpy.CStore
}

func (httpFs *httpFs) Open(fullPath string) (http.File, error) {
	log.Printf("open: %s", fullPath)
	rootHash, _, ok, err := remote.GetRoot(httpFs.c, httpFs.k)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("root missing")
	}

	ref, err := refs.GetRef(httpFs.store, rootHash)
	if err != nil {
		common.Die("error fetching ref: %s\n", err.Error())
	}

	ent, err := fs.Walk(httpFs.store, ref.Root, fullPath)
	if err != nil {
		return nil, err
	}

	if ent.EntMode.IsRegular() {
		rdr, err := fs.Open(httpFs.store, ref.Root, fullPath)
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
			root:   ref.Root,
			path:   fullPath,
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
	static.LoadFiles()

	addrArg := flag.String("addr", "127.0.0.1:8000", "address to listen on ")
	flag.Parse()

	cfg, err := common.GetConfig()
	if err != nil {
		common.Die("error getting config: %s\n", err)
	}

	k, err := common.GetKey(cfg)
	if err != nil {
		common.Die("error getting key: %s\n", err.Error())
	}

	log.Printf("connecting to remote\n")
	c, err := common.GetRemote(cfg, &k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}

	store, err := common.GetCStore(cfg, &k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	http.Handle("/", http.RedirectHandler("/browse/", http.StatusSeeOther))

	http.Handle("/browse/", http.StripPrefix("/browse", &rootHandler{
		c:     c,
		k:     &k,
		store: store,
	}))

	http.Handle("/zip/", http.StripPrefix("/zip", &zipHandler{
		c:     c,
		k:     &k,
		store: store,
	}))

	http.Handle("/raw/", http.StripPrefix("/raw", http.FileServer(&httpFs{
		c:     c,
		k:     &k,
		store: store,
	})))

	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(static.Files)))

	url := "http://" + *addrArg
	log.Printf("serving on %s\n", url)

	go func() {
		time.Sleep(200 * time.Millisecond)
		browser.OpenURL(url)
	}()

	log.Fatal(http.ListenAndServe(*addrArg, nil))
}
