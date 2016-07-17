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
	ErrTooManyPacks = errors.New("too many open pack uploads")
	ErrBadResponse  = errors.New("server sent bad response")
	ErrDisconnected = errors.New("connection disconnected")
	ErrNoSuchPid    = errors.New("no such pack id")
)

type Client struct {
	conn ReadWriteCloser

	maxMessageSizeLock sync.RWMutex
	maxMessageSize     uint32

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

func (c *Client) getMaxMessageSize() uint32 {
	c.maxMessageSizeLock.RLock()
	sz := c.maxMessageSize
	c.maxMessageSizeLock.RUnlock()
	return sz
}

func (c *Client) setMaxMessageSize(sz uint32) {
	c.maxMessageSizeLock.Lock()
	c.wLock.Lock()
	c.maxMessageSize = sz
	c.wBuf = make([]byte, sz, sz)
	c.rBuf = make([]byte, sz, sz)
	c.wLock.Unlock()
	c.maxMessageSizeLock.Unlock()
}

func readMessages(c *Client) {
	for {
		m, err := proto.ReadMessage(c.conn, c.rBuf)
		if err != nil {
			break
		}
		mid := proto.GetMessageId(m)
		if mid == proto.NOMID {
			switch m := m.(type) {
			case *proto.RPackError:
				c.setPidError(m.Pid, errors.New(m.Message))
			default:
				continue
			}
		}
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
		calls: make(map[uint16]chan proto.Message),
		fids:  make(map[uint32]struct{}),
		pids:  make(map[uint32]error),
	}
	c.setMaxMessageSize(maxsz)
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
		c.setMaxMessageSize(resp.MaxMessageSize)
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

func (c *Client) TTag(name, value string) (*proto.RTag, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TTag{
		Mid:   mid,
		Name:  name,
		Value: value,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.RTag:
		return resp, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) TCasTag(name, oldValue, newValue string) (*proto.RCasTag, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TCasTag{
		Mid:      mid,
		Name:     name,
		OldValue: oldValue,
		NewValue: newValue,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.RCasTag:
		return resp, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) TGetTag(name string) (*proto.RGetTag, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TGetTag{
		Mid:  mid,
		Name: name,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.RGetTag:
		return resp, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) TRemoveTag(name string) (*proto.RRemoveTag, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TRemoveTag{
		Mid:  mid,
		Name: name,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.RRemoveTag:
		return resp, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) TOpen(fid uint32, name string) (*proto.ROpen, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TOpen{
		Mid:  mid,
		Fid:  fid,
		Name: name,
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

func (c *Client) TWritePack(pid uint32, data []byte) error {
	return c.WriteMessage(&proto.TWritePack{
		Pid:  pid,
		Data: data,
	})
}

func (c *Client) TClosePack(pid uint32) (*proto.RClosePack, error) {
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

func (c *Client) TCancelPack(pid uint32) (*proto.RCancelPack, error) {
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TCancelPack{
		Mid: mid,
		Pid: pid,
	}, ch, mid)
	if err != nil {
		return nil, err
	}
	switch resp := resp.(type) {
	case *proto.RCancelPack:
		return resp, nil
	default:
		return nil, ErrBadResponse
	}
}

func (c *Client) nextPid() (uint32, error) {
	c.pidLock.Lock()
	defer c.pidLock.Unlock()
	pid := c.pidCount + 1
	for {
		if pid == c.pidCount {
			return 0, ErrTooManyPacks
		}
		_, ok := c.pids[pid]
		if !ok {
			c.pids[pid] = nil
			return pid, nil
		}
		pid += 1
	}
}

func (c *Client) checkPidError(pid uint32) error {
	c.pidLock.Lock()
	defer c.pidLock.Unlock()
	err, ok := c.pids[pid]
	if !ok {
		return ErrNoSuchPid
	}
	return err
}

func (c *Client) setPidError(pid uint32, err error) {
	c.pidLock.Lock()
	defer c.pidLock.Unlock()
	err, ok := c.pids[pid]
	if !ok {
		return
	}
	c.pids[pid] = err
}

func (c *Client) freePid(pid uint32) {
	c.pidLock.Lock()
	defer c.pidLock.Unlock()
	_, ok := c.pids[pid]
	if ok {
		delete(c.pids, pid)
	}
}

func (c *Client) NewPack(name string) (*Pack, error) {
	pid, err := c.nextPid()
	if err != nil {
		return nil, err
	}
	ch, mid, err := c.newCall()
	if err != nil {
		return nil, err
	}
	resp, err := c.Call(&proto.TNewPack{
		Mid:  mid,
		Pid:  pid,
		Name: name,
	}, ch, mid)
	if err != nil {
		c.freePid(pid)
		return nil, err
	}
	switch resp.(type) {
	case *proto.RNewPack:
		return &Pack{
			c:   c,
			pid: pid,
		}, nil
	default:
		c.freePid(pid)
		return nil, ErrBadResponse
	}
}

func (c *Client) Open(name string) (*File, error) {
	fid, err := c.nextFid()
	if err != nil {
		return nil, err
	}
	_, err = c.TOpen(fid, name)
	if err != nil {
		c.freeFid(fid)
		return nil, err
	}
	return &File{
		c:   c,
		fid: fid,
	}, nil
}
