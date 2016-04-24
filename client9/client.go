package client9

import (
	"acha.ninja/bpy/proto9"
	"errors"
	"io"
	"path"
	"strings"
)

var (
	ErrBadVersion = errors.New("bad version")
)

type Client struct {
	c      *proto9.Conn
	root   proto9.Fid
	maxfid proto9.Fid
	fids   map[proto9.Fid]struct{}
}

type File struct {
	c      *Client
	Iounit uint32
	Fid    proto9.Fid
	offset uint64
}

func NewClient(c *proto9.Conn) (*Client, error) {
	cl := &Client{
		c:    c,
		fids: make(map[proto9.Fid]struct{}),
	}
	return cl, cl.negotiateVersion()
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
	maxsize := uint32(65536)
	c.c.SetMaxMessageSize(maxsize)
	resp, err := c.c.Tversion(maxsize, "9P2000")
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
	c.c.SetMaxMessageSize(maxsize)
	return nil
}

func (c *Client) Attach(name, aname string) error {
	fid := c.nextFid()
	_, err := c.c.Tattach(fid, proto9.NOFID, name, aname)
	if err != nil {
		c.clunkFid(fid)
		return err
	}
	c.root = fid
	return nil
}

func (c *Client) Open(path string, mode proto9.OpenMode) (*File, error) {
	fid, err := c.walk(path)
	if err != nil {
		return nil, err
	}
	resp, err := c.c.Topen(fid, mode)
	if err != nil {
		c.clunkFid(fid)
		return nil, err
	}
	return &File{
		c:      c,
		Iounit: resp.Iounit,
	}, nil
}

func (c *Client) walk(wpath string) (proto9.Fid, error) {
	wpath = path.Clean(wpath)
	names := strings.Split(wpath, "/")
	if len(names) != 0 && names[0] == "" {
		names = names[1:]
	}
	if len(names) != 0 && names[len(names)-1] == "" {
		names = names[:len(names)-1]
	}
	fid := c.nextFid()
	resp, err := c.c.Twalk(c.root, fid, names)
	if err != nil {
		c.clunkFid(fid)
		return proto9.NOFID, err
	}
	if len(resp.Qids) != len(names) {
		c.clunkFid(fid)
		return proto9.NOFID, errors.New("walk failed")
	}
	return fid, nil
}

func (f *File) Read(buf []byte) (int, error) {
	n, err := f.ReadAt(f.offset, buf)
	f.offset += uint64(n)
	return n, err
}

func (f *File) ReadAt(offset uint64, buf []byte) (int, error) {
	n := 0
	for len(buf) != 0 {
		amnt := uint32(len(buf))
		maxamnt := f.c.c.MaxMessageSize() - proto9.ReadOverhead
		if amnt > maxamnt {
			amnt = maxamnt
		}
		resp, err := f.c.c.Tread(f.Fid, offset+uint64(n), amnt)
		if err != nil {
			return n, err
		}
		copy(buf[n:len(buf)], resp.Data)
		n += len(resp.Data)
		if uint32(len(resp.Data)) > amnt {
			return n, proto9.ErrBadResponse
		}
		if len(resp.Data) == 0 {
			return n, io.EOF
		}
	}
	return n, nil
}

func (f *File) Write(buf []byte) (int, error) {
	n, err := f.WriteAt(f.offset, buf)
	f.offset += uint64(n)
	return n, err
}

func (f *File) WriteAt(offset uint64, buf []byte) (int, error) {
	n := 0
	for len(buf) != 0 {
		amnt := uint32(len(buf))
		maxamnt := f.c.c.MaxMessageSize() - proto9.WriteOverhead
		if amnt > maxamnt {
			amnt = maxamnt
		}
		resp, err := f.c.c.Twrite(f.Fid, offset+uint64(n), buf[0:amnt])
		if err != nil {
			return n, err
		}
		buf = buf[resp.Count:]
		n += int(resp.Count)
	}
	return n, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if whence != 0 {
		return int64(f.offset), errors.New("unsupported seek")
	}
	f.offset = uint64(offset)
	return offset, nil
}

func (f *File) Close() error {
	_, err := f.c.c.Tclunk(f.Fid)
	f.c.clunkFid(f.Fid)
	return err
}
