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
	Fid    proto9.Fid
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

func (c *Client) Tversion(msize uint32, version string) (*proto9.Rversion, error) {
	msg, err := c.sendMsg(&proto9.Tversion{
		Tag:         proto9.NOTAG,
		MessageSize: msize,
		Version:     version,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rversion)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Tauth(afid proto9.Fid, uname string, aname string) (*proto9.Rauth, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Tauth{
		Tag:   tag,
		Afid:  afid,
		Uname: uname,
		Aname: aname,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rauth)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Tflush(oldtag proto9.Tag) (*proto9.Rflush, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Tflush{
		Tag:    tag,
		OldTag: oldtag,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rflush)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Tattach(fid, afid proto9.Fid, uname, aname string) (*proto9.Rattach, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Tattach{
		Tag:   tag,
		Fid:   fid,
		Afid:  afid,
		Uname: uname,
		Aname: aname,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rattach)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Twalk(fid, newfid proto9.Fid, names []string) (*proto9.Rwalk, error) {
	if len(names) > 16 {
		return nil, errors.New("cannot walk with more than 16 names")
	}
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Twalk{
		Tag:    tag,
		Fid:    fid,
		NewFid: newfid,
		Names:  names,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rwalk)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}
func (c *Client) Topen(fid proto9.Fid, mode proto9.OpenMode) (*proto9.Ropen, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Topen{
		Tag:  tag,
		Fid:  fid,
		Mode: mode,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Ropen)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Tcreate(fid proto9.Fid, name string, perm proto9.FileMode, mode proto9.OpenMode) (*proto9.Rcreate, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Tcreate{
		Tag:  tag,
		Fid:  fid,
		Name: name,
		Perm: perm,
		Mode: mode,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rcreate)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Tread(fid proto9.Fid, offset uint64, count uint32) (*proto9.Rread, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Tread{
		Tag:    tag,
		Fid:    fid,
		Offset: offset,
		Count:  count,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rread)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Twrite(fid proto9.Fid, offset uint64, buf []byte) (*proto9.Rwrite, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Twrite{
		Tag:  tag,
		Fid:  fid,
		Data: buf,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rwrite)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Tclunk(fid proto9.Fid) (*proto9.Rclunk, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Twalk{
		Tag: tag,
		Fid: fid,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rclunk)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Tremove(fid proto9.Fid) (*proto9.Rremove, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Twalk{
		Tag: tag,
		Fid: fid,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rremove)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Tstat(fid proto9.Fid) (*proto9.Rstat, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Tstat{
		Tag: tag,
		Fid: fid,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rstat)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) Twstat(fid proto9.Fid, stat proto9.Stat) (*proto9.Rwstat, error) {
	tag := c.nextTag()
	msg, err := c.sendMsg(&proto9.Twstat{
		Tag:  tag,
		Fid:  fid,
		Stat: stat,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*proto9.Rwstat)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Client) negotiateVersion() error {
	bufsz := uint32(65536)
	c.buf = make([]byte, bufsz, bufsz)
	resp, err := c.Tversion(bufsz, "9P2000")
	if err != nil {
		return err
	}
	if resp.Version != "9P2000" {
		return ErrBadVersion
	}
	c.buf = c.buf[0:resp.MessageSize]
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

func (c *Client) walk(root proto9.Fid, path string) (*File, error) {
	return nil, errors.New("unimplemented...")
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

func (f *File) Close() error {
	_, err := f.c.Tclunk(f.Fid)
	f.c.clunkFid(f.Fid)
	return err
}
