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
	c      *Client
	Iounit uint32
	fid    proto9.Fid
}

func NewClient(in io.Reader, out io.Writer) (*Client, error) {
	c := &Client{
		in:   in,
		out:  out,
		fids: make(map[proto9.Fid]struct{}),
	}
	return c, c.negotiateVersion()
}

func (c *Client) nextTag() proto9.Tag {
	panic("...")
	return 0
}

func (c *Client) nextFid() proto9.Fid {
	panic("...")
	return 0
}

func (c *Client) clunkFid(fid proto9.Fid) {
	panic("...")
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
	msg, err := proto9.UnpackMsg(c.buf[0:sz])
	if err != nil {
		return nil, err
	}
	errmsg, iserr := msg.(*proto9.Rerror)
	if iserr {
		return nil, errors.New(errmsg.Err)
	}
	return msg, nil
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

func (c *Client) Open(path string, mode proto9.OpenMode) (*File, error) {
	tag := c.nextTag()
	fid := c.nextFid()
	resp, err := c.sendMsg(&proto9.Topen{
		Tag:  tag,
		Fid:  fid,
		Mode: mode,
	})
	if err != nil {
		c.clunkFid(fid)
		return nil, err
	}
	openresp, ok := resp.(*proto9.Ropen)
	if !ok {
		return nil, ErrBadResponse
	}
	return &File{
		c:      c,
		Iounit: openresp.Iounit,
	}, nil
}

func (c *Client) walk(root proto9.Fid, path string) (*File, error) {
	return nil, errors.New("unimplemented...")
}

func (c *Client) openFid(fid proto9.Fid, mode proto9.OpenMode) (*File, error) {
	tag := c.nextTag()
	resp, err := c.sendMsg(&proto9.Topen{
		Tag:  tag,
		Fid:  fid,
		Mode: mode,
	})
	openresp, ok := resp.(*Ropen)
	if !ok {
		return nil, ErrBadResponse
	}
	return &File{
		c:      c,
		Iounit: openresp.Iounit,
	}
}

func (f *File) Read(buf []byte) (int, error) {
	n := 0
	for len(buf) {
		amnt := uint32(len(buf))
		maxamnt := uint32(len(f.c.buf) - proto9.ReadOverhead)
		if amnt > maxamnt {
			amnt = maxamnt
		}
		tag := f.c.nextTag()
		resp, err := f.c.sendMsg(&Tread{
			Tag:   tag,
			Count: amnt,
		})
		if err != nil {
			return n, err
		}
		readresp, ok := resp.(*Rread)
		if !ok {
			return n, ErrBadResponse
		}
		copy(buf[n:len(buf)], readresp)
		n += len(readresp.data)
		if len(readresp.data) > amnt {
			return n, ErrBadResponse
		}
		if len(readresp.data) == 0 {
			return n, io.Eof
		}
	}
	return n, nil
}

func (f *File) Write([]byte) (int, error) {
	return 0, errors.New("unimplemented")
}

func (f *File) Close() error {
	tag := f.c.nextTag()
	_, err := f.c.sendMsg(&Tclunk{
		Tag: tag,
		Fid: f.fid,
	})
	f.c.clunkFid(f.fid)
	return err
}
