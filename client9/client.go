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

func (c *Client) clunkFid(fid proto9.Fid) error {
	_, err := c.c.Tclunk(fid)
	c.freeFid(fid)
	return err
}

func (c *Client) freeFid(fid proto9.Fid) {
	delete(c.fids, fid)
}

func (c *Client) negotiateVersion() error {
	maxsize := uint32(131072)
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
	if resp.MessageSize < 512 {
		return proto9.ErrBuffTooSmall
	}
	c.c.SetMaxMessageSize(resp.MessageSize)
	return nil
}

func (c *Client) Attach(name, aname string) error {
	fid := c.nextFid()
	_, err := c.c.Tattach(fid, proto9.NOFID, name, aname)
	if err != nil {
		c.freeFid(fid)
		return err
	}
	c.root = fid
	return nil
}

func (c *Client) Ls(path string) ([]proto9.Stat, error) {
	f, err := c.Open(path, proto9.OREAD)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	offset := uint64(0)
	stats := make([]proto9.Stat, 0, 32)
	// We cannot use something like io.Readall to read a directory
	// because we need to ensure the read count can hold an integral number
	// of stat entries (a requirement of 9p).
	for {
		resp, err := f.Tread(offset, f.c.c.MaxMessageSize()-proto9.ReadOverhead)
		if err != nil {
			return nil, err
		}
		if len(resp.Data) == 0 {
			break
		}
		offset += uint64(len(resp.Data))
		for len(resp.Data) > 0 {
			stat := proto9.Stat{}
			statsz, err := proto9.UnpackStat(resp.Data, &stat)
			if err != nil {
				return nil, err
			}
			stats = append(stats, stat)
			resp.Data = resp.Data[statsz:]
		}
	}
	return stats, nil
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
		Fid:    fid,
		Iounit: resp.Iounit,
	}, nil
}

func (c *Client) Stat(path string) (proto9.Stat, error) {
	fid, err := c.walk(path)
	if err != nil {
		return proto9.Stat{}, err
	}
	defer c.clunkFid(fid)
	resp, err := c.c.Tstat(fid)
	if err != nil {
		return proto9.Stat{}, err
	}
	return resp.Stat, nil
}

func (c *Client) Wstat(path string, wstat proto9.Stat) error {
	fid, err := c.walk(path)
	if err != nil {
		return err
	}
	defer c.clunkFid(fid)
	_, err = c.c.Twstat(fid, wstat)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Remove(path string) error {
	fid, err := c.walk(path)
	if err != nil {
		return err
	}
	_, err = c.c.Tremove(fid)
	c.freeFid(fid)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Mkdir(fullpath string, mode proto9.FileMode) error {
	name := path.Base(fullpath)
	dir := path.Dir(fullpath)

	dirfid, err := c.walk(dir)
	if err != nil {
		return err
	}
	defer c.clunkFid(dirfid)
	fid := c.nextFid()
	_, err = c.c.Tcreate(fid, name, mode|proto9.DMDIR, proto9.ORDWR)
	if err != nil {
		return err
	}
	c.clunkFid(fid)
	return nil
}

func (c *Client) Create(fullpath string, mode proto9.FileMode, omode proto9.OpenMode) (*File, error) {
	name := path.Base(fullpath)
	dir := path.Dir(fullpath)
	if dir == "." {
		dir = "/"
	}
	fid, err := c.walk(dir)
	if err != nil {
		return nil, err
	}
	resp, err := c.c.Tcreate(fid, name, mode, proto9.ORDWR)
	if err != nil {
		c.clunkFid(fid)
		return nil, err
	}
	return &File{
		c:      c,
		Fid:    fid,
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
		c.freeFid(fid)
		return proto9.NOFID, err
	}
	if len(resp.Qids) != len(names) {
		c.freeFid(fid)
		return proto9.NOFID, errors.New("walk failed")
	}
	return fid, nil
}

func (f *File) Read(buf []byte) (int, error) {
	n, err := f.ReadAt(f.offset, buf)
	f.offset += uint64(n)
	return n, err
}

func (f *File) Tread(offset uint64, amnt uint32) (*proto9.Rread, error) {
	resp, err := f.c.c.Tread(f.Fid, offset, amnt)
	if err != nil {
		return nil, err
	}
	if uint32(len(resp.Data)) > amnt {
		return nil, proto9.ErrBadResponse
	}
	return resp, nil
}

func (f *File) ReadAt(offset uint64, buf []byte) (int, error) {
	amnt := uint32(len(buf))
	maxamnt := f.c.c.MaxMessageSize() - proto9.ReadOverhead
	if amnt > maxamnt {
		amnt = maxamnt
	}
	resp, err := f.Tread(offset, amnt)
	if err != nil {
		return 0, err
	}
	if len(resp.Data) == 0 {
		return 0, io.EOF
	}
	return copy(buf, resp.Data), nil
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
		resp, err := f.Twrite(offset+uint64(n), buf[0:amnt])
		if err != nil {
			return n, err
		}
		buf = buf[resp.Count:]
		n += int(resp.Count)
	}
	return n, nil
}

func (f *File) Twrite(offset uint64, buf []byte) (*proto9.Rwrite, error) {
	resp, err := f.c.c.Twrite(f.Fid, offset, buf)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if whence != 0 {
		return int64(f.offset), errors.New("unsupported seek")
	}
	f.offset = uint64(offset)
	return offset, nil
}

func (f *File) Close() error {
	return f.c.clunkFid(f.Fid)
}
