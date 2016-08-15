package cache

import (
	"bytes"
	"compress/flate"
	"io"
	"io/ioutil"
	"net/rpc"
)

type Client struct {
	client   *rpc.Client
	flatebuf bytes.Buffer
	flatew   *flate.Writer
}

func NewClient(rwc io.ReadWriteCloser) (*Client, error) {
	flatew, err := flate.NewWriter(ioutil.Discard, flate.BestSpeed)
	if err != nil {
		return nil, err
	}
	return &Client{
		client: rpc.NewClient(rwc),
		flatew: flatew,
	}, nil
}

func (c *Client) Get(hash [32]byte) ([]byte, bool, error) {
	val, ok, err := c.GetRaw(hash)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	rdr := flate.NewReader(bytes.NewBuffer(val))
	c.flatebuf.Reset()
	_, err = io.Copy(&c.flatebuf, rdr)
	if err != nil {
		return nil, false, err
	}
	buf := make([]byte, c.flatebuf.Len(), c.flatebuf.Len())
	copy(buf, c.flatebuf.Bytes())
	return buf, true, nil
}

func (c *Client) GetRaw(hash [32]byte) ([]byte, bool, error) {
	r := &RGet{}
	err := c.client.Call("CacheServer.Get", TGet{Hash: hash}, r)
	return r.Val, r.Ok, err
}

func (c *Client) Put(hash [32]byte, val []byte) error {
	c.flatebuf.Reset()
	c.flatew.Reset(&c.flatebuf)
	_, err := c.flatew.Write(val)
	if err != nil {
		return err
	}
	err = c.flatew.Close()
	if err != nil {
		return err
	}
	return c.PutRaw(hash, c.flatebuf.Bytes())
}

func (c *Client) PutRaw(hash [32]byte, val []byte) error {
	return c.client.Call("CacheServer.Put", TPut{Hash: hash, Val: val}, &RPut{})
}
