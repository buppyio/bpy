package cache

import (
	"net/rpc"
)

type TGet struct {
	Hash [32]byte
}

type RGet struct {
	Val []byte
	Ok  bool
}

type TPut struct {
	Hash [32]byte
	Val  []byte
}

type RPut struct {
}

type CacheServer struct {
	cache *Cache
}

func (cs *CacheServer) Get(t TGet, r *RGet) error {
	v, ok, err := cs.cache.Get(t.Hash)
	if err != nil {
		return err
	}
	r.Val = v
	r.Ok = ok
	return nil
}

func (cs *CacheServer) Put(t TPut, r *RPut) error {
	return cs.cache.Put(t.Hash, t.Val)
}

func NewServer(cache *Cache) (*rpc.Server, error) {
	cacheServer := &CacheServer{
		cache: cache,
	}
	server := rpc.NewServer()
	err := server.Register(cacheServer)
	if err != nil {
		return nil, err
	}
	return server, nil
}
