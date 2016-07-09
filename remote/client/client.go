package client

import (
	"acha.ninja/bpy/remote/proto"
	"errors"
	"io"
	"sync"
)

type ReadWriteCloser interface {
	io.Reader
	io.Writer
	io.Closer
}

var (
	ErrClientClosed = errors.New("client closed")
	ErrTooManyCalls = errors.New("too many calls in progress")
	ErrTooManyFiles = errors.New("too many open files")
	ErrBadResponse  = errors.New("server sent bad response")
	ErrDisconnected = errors.New("connection disconnected")
)

type Client struct {
	conn ReadWriteCloser

	maxMessageSize uint32

	wLock sync.Mutex
	wBuf  []byte
	rBuf  []byte

	midLock  sync.Mutex
	mIdCount uint16
	closed   bool
	calls    map[uint16]chan proto.Message

	fidLock  sync.Mutex
	fidCount uint32
	fids     map[uint32]struct{}

	pidLock  sync.Mutex
	pidCount uint32
	pids     map[uint32]error
}

func readMessages(c *Client) {
	for {
		m, err := proto.ReadMessage(c.conn, c.rBuf)
		if err != nil {
			break
		}
		mid := proto.GetMessageId(m)
		c.midLock.Lock()
		ch, ok := c.calls[mid]
		if ok {
			ch <- m
		}
		c.midLock.Unlock()
	}
	c.Close()
}

func Attach(conn ReadWriteCloser, keyId string) (*Client, error) {
	maxsz := uint32(1024 * 1024)
	c := &Client{
		conn:  conn,
		wBuf:  make([]byte, maxsz, maxsz),
		rBuf:  make([]byte, maxsz, maxsz),
		calls: make(map[uint16]chan proto.Message),
		fids:  make(map[uint32]struct{}),
		pids:  make(map[uint32]error),
	}
	go readMessages(c)

	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TAttach{
		Mid:            mid,
		MaxMessageSize: maxsz,
		Version:        "buppy1",
		KeyId:          keyId,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.RAttach:
		if resp.MaxMessageSize > maxsz || resp.Mid != 1 {
			return nil, ErrBadResponse
		}
		c.wLock.Lock()
		c.wBuf = make([]byte, resp.MaxMessageSize, resp.MaxMessageSize)
		c.wLock.Unlock()
		return c, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) Close() error {
	c.midLock.Lock()
	defer c.midLock.Unlock()
	if c.closed {
		return nil
	}
	for _, ch := range c.calls {
		close(ch)
	}
	c.closed = true
	c.conn.Close()
	return nil
}

func (c *Client) newCall() (chan proto.Message, uint16, error) {
	c.midLock.Lock()
	defer c.midLock.Unlock()

	if c.closed {
		return nil, 0, ErrDisconnected
	}

	mid := c.mIdCount + 1
	for {
		if mid == 0 {
			mid += 1
		}
		if mid == c.mIdCount {
			return nil, 0, ErrTooManyCalls
		}
		_, ok := c.calls[mid]
		if !ok {
			ch := make(chan proto.Message)
			c.calls[mid] = ch
			return ch, mid, nil
		}
		mid += 1
	}
}

func (c *Client) Call(m proto.Message, ch chan proto.Message, mid uint16) (proto.Message, error) {
	defer func() {
		c.midLock.Lock()
		delete(c.calls, mid)
		c.midLock.Unlock()
	}()
	err := c.WriteMessage(m)
	if err != nil {
		return nil, err
	}
	resp, ok := <-ch
	if !ok {
		return nil, ErrDisconnected
	}
	switch resp := resp.(type) {
	case *proto.RError:
		return nil, errors.New(resp.Message)
	default:
		return resp, nil
	}
}

func (c *Client) nextFid() (uint32, error) {
	c.fidLock.Lock()
	defer c.fidLock.Unlock()
	fid := c.fidCount + 1
	for {
		if fid == c.fidCount {
			return 0, ErrTooManyFiles
		}
		_, ok := c.fids[fid]
		if !ok {
			c.fids[fid] = struct{}{}
			return fid, nil
		}
		fid += 1
	}
}

func (c *Client) freeFid(fid uint32) {
	c.fidLock.Lock()
	defer c.fidLock.Unlock()
	_, ok := c.fids[fid]
	if ok {
		delete(c.fids, fid)
	}
}

func (c *Client) WriteMessage(m proto.Message) error {
	c.wLock.Lock()
	defer c.wLock.Unlock()
	return proto.WriteMessage(c.conn, m, c.wBuf)
}

func (c *Client) TOpen(fid uint32, path string) (*proto.ROpen, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TOpen{
		Mid:  mid,
		Fid:  fid,
		Path: path,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.ROpen:
		return resp, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) TReadAt(fid uint32, offset uint64, size uint32) (*proto.RReadAt, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TReadAt{
		Mid:    mid,
		Fid:    fid,
		Offset: offset,
		Size:   size,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.RReadAt:
		return resp, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) TClose(fid uint32) (*proto.RClose, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TClose{
		Mid: mid,
		Fid: fid,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.RClose:
		return resp, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) TNewPack(pid uint32) (*proto.RNewPack, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TNewPack{
		Mid: mid,
		Pid: pid,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.RNewPack:
		return resp, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) TPackWrite(pid uint32, data []byte) error {
	return c.WriteMessage(&proto.TPackWrite{
		Mid: proto.CASTMID,
		Pid: pid,
	})
}

func (c *Client) TClosePack(pid uint32) (*proto.TClosePack, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TClosePack{
		Mid: mid,
		Pid: pid,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.RClosePack:
		return resp, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) Open(path string) (*File, error) {
	fid, err := c.nextFid()
	if err != nil {
		return nil, err
	}
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TOpen{
		Mid:  mid,
		Path: path,
		Fid:  fid,
	}, ch, mid)
	if err != nil {
		c.freeFid(fid)
		return nil, err
	}
	switch resp.(type) {
	case *proto.ROpen:
		return &File{
			c:   c,
			fid: fid,
		}, nil
	default:
		c.freeFid(fid)
		return nil, ErrBadResponse
	}
}
