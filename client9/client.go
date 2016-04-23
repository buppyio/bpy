package client9

import (
	"acha.ninja/bpy/proto9"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

var (
	ErrBadVersion = errors.New("bad version")
)

type Client struct {
	c      *proto9.Conn
	maxfid proto9.Fid
	fids   map[proto9.Fid]struct{}
}

type File struct {
	c      *Client
	Iounit uint32
	Fid    proto9.Fid
	offset int64
}

func NewClient(c *proto9.Conn) (*Client, error) {
	c := &Client{
		c:    c,
		fids: make(map[proto9.Fid]struct{}),
	}
	return c, c.negotiateVersion()
}

func (c *Client) nextFid() proto9.Fid {
	for {
		fid := c.maxfid
		c.maxfid++
		if fid == proto9.NOFID {
			continue
		}
		_, hasfid := c.fids[fid]
		if hasfid {
			continue
		}
		c.fids[fid] = struct{}{}
		return fid
	}
}

func (c *Client) clunkFid(fid proto9.Fid) {
	delete(c.fids, fid)
}

func (c *Client) negotiateVersion() error {
	maxsize := 65536
	c.c.SetMaxMessageSize(maxsize)
	resp, err := c.Tversion(bufsz, "9P2000")
	if err != nil {
		return err
	}
	if resp.Version != "9P2000" {
		return ErrBadVersion
	}
	if resp.MessageSize > maxsize {
		return proto9.ErrBadResponse
	}
	if resp.MessageSize < 1024 {
		return proto9.ErrBuffTooSmall
	}
	c.SetMaxMessageSize(maxsize)
	return nil
}

func (c *Client) Attach() error {
	return errors.New("unimplemented")
}

func (c *Client) Open(path string, mode proto9.OpenMode) (*File, error) {
	fid := c.nextFid()
	resp, err := c.Topen(fid, mode)
	if err != nil {
		c.clunkFid(fid)
		return nil, err
	}
	return &File{
		c:      c,
		Iounit: resp.Iounit,
	}, nil
}

func (c *Client) walk(f *File, path string) (*File, error) {
	path := path.Clean(path)
	names := strings.Split(path, "/")
	if names[0] == "" {
		names = names[1:]
	}
	if names[len(names)-1] == "" {
		names = names[1:len(names)-1]
	}
	return nil, errors.New("unimplemented...")
}

func (f *File) Read(buf []byte) (int, error) {
	n, err := f.ReadAt(f.offset, buf)
	f.offset += int64(n)
	return n, err
}

func (f *File) ReadAt(offset uint64, buf []byte) (int, error) {
	n := 0
	for len(buf) != 0 {
		amnt := uint32(len(buf))
		maxamnt := uint32(len(f.c.buf) - proto9.ReadOverhead)
		if amnt > maxamnt {
			amnt = maxamnt
		}
		resp, err := f.c.Tread(f.Fid, offset+uint64(n), amnt)
		if err != nil {
			return n, err
		}
		copy(buf[n:len(buf)], resp.Data)
		n += len(resp.Data)
		if uint32(len(resp.Data)) > amnt {
			return n, ErrBadResponse
		}
		if len(resp.Data) == 0 {
			return n, io.EOF
		}
	}
	return n, nil
}

func (f *File) Write(buf []byte) (int, error) {
	n, err := f.WriteAt(f.offset, buf)
	f.offset += int64(n)
	return n, err
}

func (f *File) WriteAt(offset uint64, buf []byte) (int, error) {
	n := 0
	for len(buf) != 0 {
		amnt := uint32(len(buf))
		maxamnt := uint32(len(f.c.buf) - proto9.WriteOverhead)
		if amnt > maxamnt {
			amnt = maxamnt
		}
		resp, err := f.c.Twrite(f.Fid, offset+uint64(n), buf[0:amnt])
		if err != nil {
			return n, err
		}
		buf = buf[resp.Count:]
		n += int(resp.Count)
	}
	return n, nil
}

func (f *File) Seek(offset int64, int whence) (int, error) {
	if whence != 0 {
		return f.offset, errors.New("unsupported seek")
	}
	f.offset = offset
	return offset, nil
}

func (f *File) Close() error {
	_, err := f.c.Tclunk(f.Fid)
	f.c.clunkFid(f.Fid)
	return err
}
