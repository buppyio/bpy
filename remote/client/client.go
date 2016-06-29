package client

import (
	"errors"
	"sync"
)

var (
	ErrClientClosed = errors.New("client closed")
	ErrBadResponse = errors.New("server sent bad response")
)

type Client struct {
	lock sync.Mutex
	callCount uint16
	calls map[uint16]chan proto.Message
	closed bool
}

func (c *Client) newCall() (mid uint16, chan proto.Message, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.closed {
		return nil, errors.New("client closed")
	}
	for mid := c.callCount + 1 ; mid != c.callCount ; mid++ {
		_, ok := c.calls[mid]
		if !ok {
			ch := make(chan proto.Message, 1)
			c.calls[mid] = ch
			c.callCount = mid
			return mid, call, nil
		}
	}
	return 0, nil, errors.New("too many concurrent remote calls")
}

func (c *Client) sendMessage(m proto.Message) error {
	sz, err := m.PackedSize()
	if err != nil {
		return err
	}
	buf := make(byte[], sz, sz)
	err := m.Pack(buf)
	if err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	_, err = c.writer.Write(buf)
	if err != nil {
		c.Close()
	}
	return err
}

func (c *Client) Close() error {
	c.lock.Lock()
	c.closed = true
	for _, ch := range c.calls {
		close(ch)
	}
	c.lock.Unlock()
	return nil
}

func (c *Client) TReadAt(offset uint64) (*proto.RReadAt, error) {
	mid, resp, err := c.newCall()
	if err != nil {
		return nil, err
	}
	t := &TReadAt{
		Mid: mid,
		Offset: offset,
	}
	err = c.sendMessage(t)
	if err != nil {
		return nil, err
	}
	r, ok <- resp
	if !ok {
		return nil, ErrClientClosed
	}
	switch r := r.(type) {
	case *proto.RReadAt:
		return r, nil
	case *proto.RError:
		return nil, errors.New(r.Message)
	default:
		return nil, ErrBadResponse
	}
}
