package client

import (
	"acha.ninja/bpy/remote/proto"
	"errors"
	"io"
)

type ReadWriteCloser interface {
	io.Reader
	io.Writer
	io.Closer
}

var (
	ErrClientClosed = errors.New("client closed")
	ErrBadResponse  = errors.New("server sent bad response")
	ErrDisconnected = errors.New("connection disconnected")
)

type Client struct {
	callCount uint16
}

func Connect(conn ReadWriteCloser, keyId string) (*Client, error) {
	maxsz := 1024 * 1024
	buf := make([]byte, maxsz, maxsz)

	err := c.writeMessage(&proto.TAttach{
		Mid:            1,
		MaxMessageSize: maxsz,
		Version:        "buppy1",
		KeyId:          keyId,
	})
	if err != nil {
		return nil, err
	}

	resp, err := proto.ReadMessage()
	if err != nil {
		return nil, err
	}

	switch resp := resp.(type) {
	case *proto.RAttach:
		if resp.MaxMessageSize > maxsz || resp.Mid != 1 {
			return nil, ErrBadResponse
		}
		buf = make([]byte, resp.MaxMessageSize, resp.MaxMessageSize)
	case *RError:
		if resp.Mid != 1 {
			return nil, ErrBadResponse
		}
		return nil, errors.New(resp.Message)
	default:
		return nil, ErrBadResponse
	}

	return &Client{
		conn: conn,
		buf:  buf,
	}
}

func (c *Client) WriteMessage(m proto.Message) error {
	return errors.New("unimpl")
}

func (c *Client) nextMid() uint16 {
	c.callCount += 1
	if c.callCount == proto.CASTMID {
		c.callCount += 1
	}
	return c.callCount
}

func (c *Client) TOpen(path string) (*proto.ROpen, error) {
	mid := c.nextMid()
	err := c.writeMessage(&proto.TOpen{
		Mid:  mid,
		Path: path,
	})
	resp, err := c.readMessage()
	if err != nil {
		return nil, err
	}
	if proto.GetMessageId(resp) != mid {
		return nil, ErrBadResponse
	}
	switch resp := resp.(type) {
	case *proto.ROpen:
		return resp, nil
	case *proto.RError:
		return nil, errors.New(resp.Message)
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) TReadAt(fid uint64, offset uint64) (*proto.ROpen, error) {
	mid := c.nextMid()
	err := c.writeMessage(&proto.TOpen{
		Mid:    mid,
		Fid:    fid,
		Offset: offset,
	})
	resp, err := c.readMessage()
	if err != nil {
		return nil, err
	}
	if proto.GetMessageId(resp) != mid {
		return nil, ErrBadResponse
	}
	switch resp := resp.(type) {
	case *proto.RReadAt:
		return resp, nil
	case *proto.RError:
		return nil, errors.New(resp.Message)
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) TReadAt(fid uint64, offset uint64) (*proto.ROpen, error) {
	mid := c.nextMid()
	err := c.writeMessage(&proto.TOpen{
		Mid:    mid,
		Fid:    fid,
		Offset: offset,
	})
	resp, err := c.readMessage()
	if err != nil {
		return nil, err
	}
	if proto.GetMessageId(resp) != mid {
		return nil, ErrBadResponse
	}
	switch resp := resp.(type) {
	case *proto.RReadAt:
		return resp, nil
	case *proto.RError:
		return nil, errors.New(resp.Message)
	default:
		return nil, ErrBadResponse
	}
}
