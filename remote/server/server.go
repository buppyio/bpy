package server

import (
	"acha.ninja/bpy/remote/proto"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

var (
	ErrBadRequest         = errors.New("bad request")
	ErrPidInUse           = errors.New("pid in use")
	ErrFidInUse           = errors.New("fid in use")
	ErrNoSuchPid          = errors.New("no such pid")
	ErrNoSuchFid          = errors.New("no such fid")
	ErrNoSuchTag          = errors.New("no such tag")
	ErrTagAlreadyExists   = errors.New("tag already exists")
	ErrStaleTagValue      = errors.New("tag value stale (concurrent write?)")
	ErrGeneratingPackName = errors.New("error generating pack name")
)

const (
	TagBucketName = "tags"
	TagDBName     = "tags.db"
)

type ReadWriteCloser interface {
	io.Reader
	io.Writer
	io.Closer
}

type file interface {
	// Semantics like io.Reader, but with interface like io.ReaderAt
	ReadAtOffset([]byte, uint64) (int, error)
	io.Closer
}

type uploadState struct {
	tmpPath string
	path    string
	err     error
	file    *os.File
}

type server struct {
	servePath string
	tagDBPath string
	buf       []byte
	fids      map[uint32]file
	pids      map[uint32]*uploadState
}

type osfile struct {
	f *os.File
}

func (f *osfile) ReadAtOffset(buf []byte, offset uint64) (int, error) {
	return f.f.ReadAt(buf, int64(offset))
}

func (f *osfile) Close() error {
	return f.f.Close()
}

func makeError(mid uint16, err error) proto.Message {
	return &proto.RError{
		Mid:     mid,
		Message: err.Error(),
	}
}

func (srv *server) handleTOpen(t *proto.TOpen) proto.Message {
	_, ok := srv.fids[t.Fid]
	if ok {
		return makeError(t.Mid, ErrFidInUse)
	}

	if t.Name == "packs" {
		srv.fids[t.Fid] = &packListingFile{
			packDir: filepath.Join(srv.servePath, "packs"),
		}
		return &proto.ROpen{
			Mid: t.Mid,
		}
	}

	if t.Name == "tags" {
		srv.fids[t.Fid] = &tagListingFile{
			tagDBPath: srv.tagDBPath,
		}
		return &proto.ROpen{
			Mid: t.Mid,
		}
	}

	matched, err := regexp.MatchString("packs/[a-zA-Z0-9\\.]+", t.Name)
	if err != nil || !matched {
		return makeError(t.Mid, ErrBadRequest)
	}
	fpath := path.Join(srv.servePath, t.Name)
	f, err := os.Open(fpath)
	if err != nil {
		return makeError(t.Mid, err)
	}
	srv.fids[t.Fid] = &osfile{f: f}
	return &proto.ROpen{
		Mid: t.Mid,
	}
}

func (srv *server) handleTReadAt(t *proto.TReadAt) proto.Message {
	f, ok := srv.fids[t.Fid]
	if !ok {
		return makeError(t.Mid, ErrNoSuchFid)
	}
	if t.Size+proto.READOVERHEAD > uint32(len(srv.buf)) {
		return makeError(t.Mid, ErrBadRequest)
	}
	buf := make([]byte, t.Size, t.Size)
	n, err := f.ReadAtOffset(buf, t.Offset)
	if err != nil && err != io.EOF {
		return makeError(t.Mid, err)
	}
	return &proto.RReadAt{
		Mid:  t.Mid,
		Data: buf[:n],
	}
}

func (srv *server) handleTClose(t *proto.TClose) proto.Message {
	f, ok := srv.fids[t.Fid]
	if !ok {
		return makeError(t.Mid, ErrNoSuchFid)
	}
	f.Close()
	delete(srv.fids, t.Fid)
	return &proto.RClose{
		Mid: t.Mid,
	}
}

func (srv *server) handleTNewPack(t *proto.TNewPack) proto.Message {
	_, ok := srv.pids[t.Pid]
	if ok {
		return makeError(t.Mid, ErrPidInUse)
	}
	matched, err := regexp.MatchString("packs/[a-zA-Z0-9]+", t.Name)
	if err != nil || !matched {
		return makeError(t.Mid, ErrBadRequest)
	}
	name := path.Join(srv.servePath, t.Name)
	tmpPath := name + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return makeError(t.Mid, fmt.Errorf("cannot create temporary packfile: %s", err.Error()))
	}
	srv.pids[t.Pid] = &uploadState{
		tmpPath: tmpPath,
		path:    name,
		file:    f,
	}
	return &proto.RNewPack{
		Mid: t.Mid,
	}
}

func (srv *server) handleTWritePack(t *proto.TWritePack) proto.Message {
	state, ok := srv.pids[t.Pid]
	if !ok {
		return &proto.RPackError{
			Pid:     t.Pid,
			Message: ErrNoSuchPid.Error(),
		}
	}
	if state.err != nil {
		return &proto.RPackError{
			Pid:     t.Pid,
			Message: state.err.Error(),
		}
	}
	_, err := state.file.Write(t.Data)
	if err != nil {
		return &proto.RPackError{
			Pid:     t.Pid,
			Message: err.Error(),
		}
	}
	return nil
}

func (srv *server) handleTClosePack(t *proto.TClosePack) proto.Message {
	state, ok := srv.pids[t.Pid]
	if !ok {
		return makeError(t.Mid, ErrNoSuchPid)
	}
	delete(srv.pids, t.Pid)
	if state.err != nil {
		state.file.Close()
		return makeError(t.Mid, state.err)
	}
	err := state.file.Close()
	if err != nil {
		return makeError(t.Mid, err)
	}
	err = os.Rename(state.tmpPath, state.path)
	if err != nil {
		return makeError(t.Mid, err)
	}
	return &proto.RClosePack{
		Mid: t.Mid,
	}
}

func (srv *server) handleTCancelPack(t *proto.TCancelPack) proto.Message {
	state, ok := srv.pids[t.Pid]
	if !ok {
		return makeError(t.Mid, ErrNoSuchPid)
	}
	delete(srv.pids, t.Pid)
	err := state.file.Close()
	if err != nil {
		return makeError(t.Mid, err)
	}
	err = os.Remove(state.tmpPath)
	if err != nil {
		return makeError(t.Mid, err)
	}
	return &proto.RCancelPack{
		Mid: t.Mid,
	}
}

func (srv *server) handleTTag(t *proto.TTag) proto.Message {
	db, err := openTagDB(srv.tagDBPath)
	if err != nil {
		makeError(t.Mid, err)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TagBucketName))
		valueBytes := b.Get([]byte(t.Name))
		if valueBytes != nil {
			return ErrTagAlreadyExists
		}
		err := b.Put([]byte(t.Name), []byte(t.Value))
		return err
	})
	if err != nil {
		return makeError(t.Mid, err)
	}
	return &proto.RTag{
		Mid: t.Mid,
	}
}

func (srv *server) handleTGetTag(t *proto.TGetTag) proto.Message {
	db, err := openTagDB(srv.tagDBPath)
	if err != nil {
		makeError(t.Mid, err)
	}
	defer db.Close()
	var value string
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TagBucketName))
		valueBytes := b.Get([]byte(t.Name))
		if valueBytes == nil {
			return ErrNoSuchTag
		}
		value = string(valueBytes)
		return nil
	})
	if err != nil {
		return makeError(t.Mid, err)
	}
	return &proto.RGetTag{
		Mid:   t.Mid,
		Value: value,
	}
}

func (srv *server) handleTRemoveTag(t *proto.TRemoveTag) proto.Message {
	db, err := openTagDB(srv.tagDBPath)
	if err != nil {
		makeError(t.Mid, err)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TagBucketName))
		valueBytes := b.Get([]byte(t.Name))
		if string(valueBytes) != t.OldValue {
			return ErrStaleTagValue
		}
		err := b.Delete([]byte(t.Name))
		return err
	})
	if err != nil {
		return makeError(t.Mid, err)
	}
	return &proto.RRemoveTag{
		Mid: t.Mid,
	}
}

func handleAttach(conn ReadWriteCloser, root string) (*server, error) {
	maxsz := uint32(1024 * 1024)
	buf := make([]byte, maxsz, maxsz)

	t, err := proto.ReadMessage(conn, buf)
	if err != nil {
		return nil, err
	}

	switch t := t.(type) {
	case *proto.TAttach:
		if t.Mid != 1 || t.Version != "buppy1" {
			return nil, ErrBadRequest
		}
		if t.MaxMessageSize < maxsz {
			maxsz = t.MaxMessageSize
		}
		buf = buf[:maxsz]
		err = proto.WriteMessage(conn, &proto.RAttach{
			Mid:            t.Mid,
			MaxMessageSize: maxsz,
		}, buf)
		if err != nil {
			return nil, ErrBadRequest
		}

		matched, err := regexp.MatchString("[a-zA-Z0-9]+", t.KeyId)
		if err != nil || !matched {
			return nil, ErrBadRequest
		}
		servePath := filepath.Join(root, t.KeyId)
		err = os.MkdirAll(filepath.Join(servePath, "packs"), 0777)
		if err != nil {
			return nil, err
		}
		return &server{
			servePath: servePath,
			tagDBPath: filepath.Join(servePath, TagDBName),
			buf:       buf,
			fids:      make(map[uint32]file),
			pids:      make(map[uint32]*uploadState),
		}, nil
	default:
		return nil, ErrBadRequest
	}
}

func Serve(conn ReadWriteCloser, root string) error {
	defer conn.Close()

	srv, err := handleAttach(conn, root)
	if err != nil {
		return err
	}
	for {
		var r proto.Message

		t, err := proto.ReadMessage(conn, srv.buf)
		if err != nil {
			return err
		}
		log.Printf("t=%#v", t)
		switch t := t.(type) {
		case *proto.TOpen:
			r = srv.handleTOpen(t)
		case *proto.TNewPack:
			r = srv.handleTNewPack(t)
		case *proto.TWritePack:
			r = srv.handleTWritePack(t)
		case *proto.TClosePack:
			r = srv.handleTClosePack(t)
		case *proto.TCancelPack:
			r = srv.handleTCancelPack(t)
		case *proto.TReadAt:
			r = srv.handleTReadAt(t)
		case *proto.TClose:
			r = srv.handleTClose(t)
		case *proto.TTag:
			r = srv.handleTTag(t)
		case *proto.TGetTag:
			r = srv.handleTGetTag(t)
		case *proto.TRemoveTag:
			r = srv.handleTRemoveTag(t)
		default:
			return ErrBadRequest
		}
		if r != nil {
			log.Printf("r=%#v", r)
			err = proto.WriteMessage(conn, r, srv.buf)
			if err != nil {
				return err
			}
		}
	}
}
