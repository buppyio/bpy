package proto9

import (
	"errors"
	"io"
)

var (
	ErrBadResponse = errors.New("bad response")
)

type IOError struct {
	Err error
}

func (e *IOError) Error() string {
	return e.Err.Error()
}

type Conn struct {
	maxtag Tag
	tags   map[Tag]struct{}
	in     io.Reader
	out    io.Writer
	buf    []byte
}

func NewConn(in io.Reader, out io.Writer, sz uint32) *Conn {
	c := &Conn{
		in:   in,
		out:  out,
		tags: make(map[Tag]struct{}),
	}
	c.SetMaxMessageSize(sz)
	return c
}

func (c *Conn) MaxMessageSize() uint32 {
	return uint32(len(c.buf))
}

func (c *Conn) SetMaxMessageSize(sz uint32) {
	c.buf = make([]byte, sz, sz)
}

func (c *Conn) nextTag() Tag {
	for {
		tag := c.maxtag
		c.maxtag++
		if tag == NOTAG {
			continue
		}
		_, hastag := c.tags[tag]
		if hastag {
			continue
		}
		c.tags[tag] = struct{}{}
		return tag
	}
}

func (c *Conn) clunkTag(tag Tag) {
	delete(c.tags, tag)
}

func (c *Conn) sendMsg(msg Msg) (Msg, error) {
	err := WriteMsg(c.out, c.buf, msg)
	if err != nil {
		return nil, &IOError{Err: err}
	}
	resp, err := ReadMsg(c.in, c.buf)
	if err != nil {
		return nil, &IOError{Err: err}
	}
	eresp, iserr := resp.(*Rerror)
	if iserr {
		return nil, errors.New(eresp.Err)
	}
	return resp, nil
}

func (c *Conn) Tversion(msize uint32, version string) (*Rversion, error) {
	msg, err := c.sendMsg(&Tversion{
		Tag:         NOTAG,
		MessageSize: msize,
		Version:     version,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rversion)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Tauth(afid Fid, uname string, aname string) (*Rauth, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Tauth{
		Tag:   tag,
		Afid:  afid,
		Uname: uname,
		Aname: aname,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rauth)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Tflush(oldtag Tag) (*Rflush, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Tflush{
		Tag:    tag,
		OldTag: oldtag,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rflush)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Tattach(fid, afid Fid, uname, aname string) (*Rattach, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Tattach{
		Tag:   tag,
		Fid:   fid,
		Afid:  afid,
		Uname: uname,
		Aname: aname,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rattach)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Twalk(fid, newfid Fid, names []string) (*Rwalk, error) {
	if len(names) > 16 {
		return nil, errors.New("cannot walk with more than 16 names")
	}
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Twalk{
		Tag:    tag,
		Fid:    fid,
		NewFid: newfid,
		Names:  names,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rwalk)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}
func (c *Conn) Topen(fid Fid, mode OpenMode) (*Ropen, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Topen{
		Tag:  tag,
		Fid:  fid,
		Mode: mode,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Ropen)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Tcreate(fid Fid, name string, perm FileMode, mode OpenMode) (*Rcreate, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Tcreate{
		Tag:  tag,
		Fid:  fid,
		Name: name,
		Perm: perm,
		Mode: mode,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rcreate)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Tread(fid Fid, offset uint64, count uint32) (*Rread, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Tread{
		Tag:    tag,
		Fid:    fid,
		Offset: offset,
		Count:  count,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rread)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Twrite(fid Fid, offset uint64, buf []byte) (*Rwrite, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Twrite{
		Tag:  tag,
		Fid:  fid,
		Data: buf,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rwrite)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Tclunk(fid Fid) (*Rclunk, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Tclunk{
		Tag: tag,
		Fid: fid,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rclunk)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Tremove(fid Fid) (*Rremove, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Twalk{
		Tag: tag,
		Fid: fid,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rremove)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Tstat(fid Fid) (*Rstat, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Tstat{
		Tag: tag,
		Fid: fid,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rstat)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}

func (c *Conn) Twstat(fid Fid, stat Stat) (*Rwstat, error) {
	tag := c.nextTag()
	defer c.clunkTag(tag)
	msg, err := c.sendMsg(&Twstat{
		Tag:  tag,
		Fid:  fid,
		Stat: stat,
	})
	if err != nil {
		return nil, err
	}
	resp, ok := msg.(*Rwstat)
	if !ok {
		return nil, ErrBadResponse
	}
	return resp, nil
}
