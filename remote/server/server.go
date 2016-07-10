package server

import (
	"acha.ninja/bpy/remote/proto"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

var (
	ErrBadRequest         = errors.New("bad request")
	ErrPidInUse           = errors.New("pid in use")
	ErrNoSuchPid          = errors.New("no such pid")
	ErrGeneratingPackName = errors.New("error generating pack name")
)

type ReadWriteCloser interface {
	io.Reader
	io.Writer
	io.Closer
}

type file interface {
}

type uploadState struct {
	tmpPath string
	path    string
	err     error
	file    *os.File
}

type server struct {
	servePath string
	buf       []byte
	fids      map[uint32]file
	pids      map[uint32]*uploadState
}

func makeError(mid uint16, err error) proto.Message {
	return &proto.RError{
		Mid:     mid,
		Message: err.Error(),
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
		_, err = os.Stat(servePath)
		if os.IsNotExist(err) {
			err = os.Mkdir(servePath, 0777)
		}
		if err != nil {
			return nil, err
		}
		return &server{
			servePath: servePath,
			buf:       buf,
			fids:      make(map[uint32]file),
			pids:      make(map[uint32]*uploadState),
		}, nil
	default:
		return nil, ErrBadRequest
	}
}

func (srv *server) handleTOpen(t *proto.TOpen) proto.Message {
	return makeError(t.Mid, errors.New("unimplemented"))
}

func (srv *server) handleTNewPack(t *proto.TNewPack) proto.Message {
	_, ok := srv.pids[t.Pid]
	if ok {
		return makeError(t.Mid, ErrPidInUse)
	}
	randBuf := [32]byte{}
	_, err := io.ReadFull(rand.Reader, randBuf[:])
	if err != nil {
		return makeError(t.Mid, ErrGeneratingPackName)
	}
	name := filepath.Join(srv.servePath, hex.EncodeToString(randBuf[:]))
	tmpPath := name + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return makeError(t.Mid, fmt.Errorf("cannot create temporary packfile: %s", err.Error()))
	}
	srv.pids[t.Pid] = &uploadState{
		tmpPath: tmpPath,
		path:    name + ".ebpack",
		file:    f,
	}
	return &proto.RNewPack{
		Mid: t.Mid,
	}
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
		case *proto.TClosePack:
			r = srv.handleTClosePack(t)
		default:
			return ErrBadRequest
		}
		log.Printf("r=%#v", r)
		if r != nil {
			err = proto.WriteMessage(conn, r, srv.buf)
			if err != nil {
				return err
			}
		}
	}
}
