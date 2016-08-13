package cache

import (
	"io"
	"net/rpc"
)

type Client struct {
	client *rpc.Client
}

func NewClient(rwc io.ReadWriteCloser) (*Client, error) {
	return &Client{
		client: rpc.NewClient(rwc),
	}, nil
}

func (c *Client) Get(hash [32]byte) ([]byte, bool, error) {
	r := &RGet{}
	err := c.client.Call("CacheServer.Get", TGet{Hash: hash}, r)
	return r.Val, r.Ok, err
}

func (c *Client) Put(hash [32]byte, val []byte) error {
	r := &RPut{}
	return c.client.Call("CacheServer.Put", TPut{Hash: hash, Val: val}, r)
}
