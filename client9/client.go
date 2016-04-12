package client9

import (
	"acha.ninja/bpy/proto9"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	ErrBadResponse = errors.New("bad response")
	ErrBadVersion  = errors.New("bad version")
)

type Client struct {
	maxfid proto9.Fid
	fids   map[proto9.Fid]struct{}
	in     io.Reader
	out    io.Writer
	buf    []byte
}

type File struct {
	c   *Client
	fid proto9.Fid
}

func NewClient(in io.Reader, out io.Writer) (*Client, error) {
	c := &Client{
		in:   in,
		out:  out,
		fids: make(map[proto9.Fid]struct{}),
	}
	return c, c.negotiateVersion()
}

func (c *Client) sendMsg(msg proto9.Msg) (proto9.Msg, error) {
	packed, err := proto9.PackMsg(c.buf, msg)
	if err != nil {
		return nil, err
	}
	_, err = c.out.Write(packed)
	if err != nil {
		return nil, err
	}
	resp, err := c.readMsg()
	if err != nil {
		return nil, err
	}
	errmsg, ok := resp.(*proto9.Rerror)
	if ok {
		return nil, fmt.Errorf("remote error: %s", errmsg.Err)
	}
	return resp, nil
}

func (c *Client) readMsg() (proto9.Msg, error) {
	if len(c.buf) < 5 {
		return nil, proto9.ErrBuffTooSmall
	}
	_, err := c.in.Read(c.buf[0:5])
	if err != nil {
		return nil, err
	}
	sz := int(binary.LittleEndian.Uint16(c.buf[0:4]))
	if len(c.buf) < sz {
		return nil, proto9.ErrBuffTooSmall
	}
	_, err = c.in.Read(c.buf[5:sz])
	if err != nil {
		return nil, err
	}
	return proto9.UnpackMsg(c.buf[0:sz])
}

func (c *Client) negotiateVersion() error {
	bufsz := uint32(65536)
	c.buf = make([]byte, bufsz, bufsz)
	resp, err := c.sendMsg(&proto9.Tversion{
		Tag:         proto9.NOTAG,
		Version:     "9P2000",
		MessageSize: bufsz,
	})
	if err != nil {
		return err
	}
	rver, ok := resp.(*proto9.Rversion)
	if !ok {
		return ErrBadResponse
	}
	if rver.Version != "9P2000" {
		return ErrBadVersion
	}
	c.buf = c.buf[0:rver.MessageSize]
	return nil
}

func (c *Client) Attach() error {
	return errors.New("unimplemented")
}

func (c *Client) Open(path string) (*File, error) {
	return nil, errors.New("unimplemented")
}

func (f *File) Read([]byte) (int, error) {
	return 0, errors.New("unimplemented")
}

func (f *File) Write([]byte) (int, error) {
	return 0, errors.New("unimplemented")
}

func (f *File) Close() error {
	return errors.New("unimplemented")
}
