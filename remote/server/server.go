package server

import (
	"acha.ninja/bpy/remote/proto"
	"errors"
	"io"
	"log"
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
	buf       []byte
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
		return &server{
			servePath: filepath.Join(root, t.KeyId),
			buf:       buf,
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
