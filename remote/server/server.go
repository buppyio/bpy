package server

import (
	"errors"
	"fmt"
	"github.com/buppyio/bpy/drive"
	"github.com/buppyio/bpy/remote/proto"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	ErrBadRequest         = errors.New("bad request")
	ErrServerError        = errors.New("server error")
	ErrPidInUse           = errors.New("pid in use")
	ErrFidInUse           = errors.New("fid in use")
	ErrNoSuchPid          = errors.New("no such pid")
	ErrNoSuchFid          = errors.New("no such fid")
	ErrWrongKeyId         = errors.New("attaching with wrong key")
	ErrBadGCID            = errors.New("GCID for remove incorrect (another concurrent gc)?")
	ErrGCInProgress       = errors.New("gc in progress")
	ErrStaleRootValue     = errors.New("root value stale (concurrent write?)")
	ErrGeneratingPackName = errors.New("error generating pack name")
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
	tmpPath  string
	path     string
	packName string
	err      error
	file     *os.File
}

type server struct {
	servePath string
	drive     *drive.Drive
	keyId     string
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
		tmpPath:  tmpPath,
		path:     name,
		packName: t.Name,
		file:     f,
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
	err := state.file.Sync()
	if err != nil {
		return makeError(t.Mid, err)
	}
	err = state.file.Close()
	if err != nil {
		return makeError(t.Mid, err)
	}
	err = os.Rename(state.tmpPath, state.path)
	if err != nil {
		return makeError(t.Mid, err)
	}
	err = srv.drive.AddPack(state.packName, t.GCGeneration)
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

func (srv *server) handleTGetRoot(t *proto.TGetRoot) proto.Message {
	value, version, signature, err := srv.drive.GetRoot()

	if err != nil {
		return makeError(t.Mid, err)
	}

	return &proto.RGetRoot{
		Mid:       t.Mid,
		Value:     value,
		Signature: signature,
		Version:   version,
		Ok:        value != "",
	}
}

func (srv *server) handleTCasRoot(t *proto.TCasRoot) proto.Message {
	ok, err := srv.drive.CasRoot(t.Value, t.Version, t.Signature, t.Generation)
	if err != nil {
		return makeError(t.Mid, err)
	}
	return &proto.RCasRoot{
		Mid: t.Mid,
		Ok:  ok,
	}
}

func (srv *server) handleTRemove(t *proto.TRemove) proto.Message {
	matched, err := regexp.MatchString("packs/[a-zA-Z0-9\\.]+", t.Path)
	if err != nil || !matched {
		return makeError(t.Mid, ErrBadRequest)
	}
	err = srv.drive.RemovePack(t.Path, t.GCGeneration)
	if err != nil || !matched {
		return makeError(t.Mid, err)
	}
	return &proto.RRemove{
		Mid: t.Mid,
	}
}

func (srv *server) handleTStartGC(t *proto.TStartGC) proto.Message {
	err := srv.drive.StartGC()
	if err != nil {
		return makeError(t.Mid, err)
	}
	return &proto.RStartGC{
		Mid: t.Mid,
	}
}

func (srv *server) handleTStopGC(t *proto.TStopGC) proto.Message {
	err := srv.drive.StopGC()
	if err != nil {
		return makeError(t.Mid, err)
	}
	return &proto.RStopGC{
		Mid: t.Mid,
	}
}

func (srv *server) handleTGetGeneration(t *proto.TGetGeneration) proto.Message {
	gen, err := srv.drive.GetGCGeneration()

	if err != nil {
		return makeError(t.Mid, err)
	}
	return &proto.RGetGeneration{
		Mid:        t.Mid,
		Generation: gen,
	}
}

func cleanOldTempPacks(packPath string) error {
	dirEnts, err := ioutil.ReadDir(packPath)
	if err != nil {
		return err
	}
	for _, ent := range dirEnts {
		if !strings.HasSuffix(ent.Name(), ".tmp") {
			continue
		}
		if !(time.Now().Sub(ent.ModTime()).Hours() > 7*24) {
			continue
		}
		tmpFilePath := filepath.Join(packPath, ent.Name())
		err = os.Remove(tmpFilePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (srv *server) handleTAttach(t *proto.TAttach) proto.Message {
	if t.Mid != 1 || t.Version != "buppy1" {
		return makeError(t.Mid, ErrBadRequest)
	}
	maxsz := uint32(len(srv.buf))
	if t.MaxMessageSize < maxsz {
		maxsz = t.MaxMessageSize
	}
	srv.buf = srv.buf[:maxsz]
	matched, err := regexp.MatchString("[a-zA-Z0-9]+", t.KeyId)
	if err != nil || !matched {
		return makeError(t.Mid, ErrBadRequest)
	}

	ok, err := srv.drive.Attach(t.KeyId)
	if err != nil {
		return makeError(t.Mid, err)
	}

	if !ok {
		return makeError(t.Mid, ErrWrongKeyId)
	}

	srv.keyId = t.KeyId

	packPath := filepath.Join(srv.servePath, "packs")
	err = os.MkdirAll(packPath, 0777)
	if err != nil {
		return makeError(t.Mid, err)
	}
	// XXX This needs to do gc
	err = cleanOldTempPacks(packPath)
	if err != nil {
		return makeError(t.Mid, err)
	}
	return &proto.RAttach{
		Mid:            t.Mid,
		MaxMessageSize: maxsz,
	}
}

func (srv *server) awaitAttach(conn ReadWriteCloser) error {

	t, err := proto.ReadMessage(conn, srv.buf)
	if err != nil {
		return err
	}
	switch t := t.(type) {
	case *proto.TAttach:
		r := srv.handleTAttach(t)
		err = proto.WriteMessage(conn, r, srv.buf)
		if err != nil {
			return err
		}
		_, iserr := r.(*proto.RError)
		if iserr {
			return ErrBadRequest
		}
		return nil
	default:
		return ErrBadRequest
	}
}

func Serve(conn ReadWriteCloser, root string) error {
	defer conn.Close()

	maxsz := uint32(1024 * 1024)
	d, err := drive.Open(filepath.Join(root, "drive.db"))
	if err != nil {
		return err
	}
	srv := &server{
		servePath: root,
		drive:     d,
		buf:       make([]byte, maxsz, maxsz),
		fids:      make(map[uint32]file),
		pids:      make(map[uint32]*uploadState),
	}

	err = srv.awaitAttach(conn)
	if err != nil {
		return err
	}

	for {
		var r proto.Message

		t, err := proto.ReadMessage(conn, srv.buf)
		if err != nil {
			return err
		}
		// log.Printf("t=%#v", t)
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
		case *proto.TGetRoot:
			r = srv.handleTGetRoot(t)
		case *proto.TCasRoot:
			r = srv.handleTCasRoot(t)
		case *proto.TRemove:
			r = srv.handleTRemove(t)
		case *proto.TStartGC:
			r = srv.handleTStartGC(t)
		case *proto.TStopGC:
			r = srv.handleTStopGC(t)
		case *proto.TGetGeneration:
			r = srv.handleTGetGeneration(t)
		default:
			return ErrBadRequest
		}
		if r != nil {
			// log.Printf("r=%#v", r)
			err = proto.WriteMessage(conn, r, srv.buf)
			if err != nil {
				return err
			}
		}
	}
}
