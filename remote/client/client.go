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
	ErrBadResponse  = errors.New("server sent bad response")
	ErrDisconnected = errors.New("connection disconnected")
)

type Client struct {
	conn ReadWriteCloser

	wLock sync.Mutex
	wBuf  []byte
	rBuf  []byte

	midLock  sync.Mutex
	mIdCount uint16
	closed   bool
	calls    map[uint16]chan proto.Message
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
	c.midLock.Lock()
	for _, ch := range c.calls {
		close(ch)
	}
	c.closed = true
	c.midLock.Unlock()
}

func Attach(conn ReadWriteCloser, keyId string) (*Client, error) {
	maxsz := uint32(1024 * 1024)
	c := &Client{
		conn: conn,
		wBuf: make([]byte, maxsz, maxsz),
		rBuf: make([]byte, maxsz, maxsz),
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
		if ok {
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

func (c *Client) WriteMessage(m proto.Message) error {
	c.wLock.Lock()
	defer c.wLock.Unlock()
	return proto.WriteMessage(c.conn, m, c.wBuf)
}

/*

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
*/
