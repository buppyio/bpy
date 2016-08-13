package cstore

import (
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cstore/cache"
)

type CachedCStore struct {
	store bpy.CStore
	cache *cache.Client
}

func NewCachedCStore(store bpy.CStore, cache *cache.Client) bpy.CStore {
	return &CachedCStore{
		cache: cache,
		store: store,
	}
}

func (c *CachedCStore) Get(hash [32]byte) ([]byte, error) {
	v, ok, err := c.cache.Get(hash)
	if err != nil {
		return nil, err
	}
	if ok {
		return v, nil
	}
	v, err = c.store.Get(hash)
	if err != nil {
		return nil, err
	}
	err = c.cache.Put(hash, v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (c *CachedCStore) Put(val []byte) ([32]byte, error) {
	hash, err := c.store.Put(val)
	if err != nil {
		return [32]byte{}, err
	}
	err = c.cache.Put(hash, val)
	if err != nil {
		return [32]byte{}, err
	}
	return hash, nil
}

func (c *CachedCStore) Flush() error {
	return c.store.Flush()
}

func (c *CachedCStore) Close() error {
	return c.store.Close()
}
