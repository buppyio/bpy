package server

import (
	"acha.ninja/bpy/remote/proto"
	"errors"
	"io"
	"path/filepath"
	"regexp"
)

var (
	ErrBadRequest = errors.New("bad request")
)

type ReadWriteCloser interface {
	io.Reader
	io.Writer
	io.Closer
}

type File interface {
}

type server struct {
	servePath string
	fids      map[uint16]File
}

func (srv *server) handleTOpen(t *proto.TOpen) proto.Message {
	return makeError(t.Mid, errors.New("unimplemented"))
}

func makeError(mid uint16, err error) proto.Message {
	return &proto.RError{
		Mid:     mid,
		Message: err.Error(),
	}
}

func Serve(conn ReadWriteCloser, root string) {
	var srv *server

	maxsz := uint32(1024 * 1024)
	buf := make([]byte, maxsz, maxsz)

	t, err := proto.ReadMessage(conn, buf)
	if err != nil {
		return
	}

	switch t := t.(type) {
	case *proto.TAttach:
		if t.Mid != 1 || t.Version != "buppy1" {
			return
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
			return
		}

		matched, err := regexp.MatchString("[a-zA-Z0-9]+", t.KeyId)
		if err != nil || !matched {
			return
		}
		srv = &server{
			servePath: filepath.Join(root, t.KeyId),
		}
	default:
		return
	}

	for {
		var resp proto.Message

		t, err := proto.ReadMessage(conn, buf)
		if err != nil {
			break
		}
		switch t := t.(type) {
		case *proto.TOpen:
			resp = srv.handleTOpen(t)
		default:
			return
		}
		err = proto.WriteMessage(conn, resp, buf)
		if err != nil {
			break
		}
	}
}
