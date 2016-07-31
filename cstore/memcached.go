package cstore

import (
	"acha.ninja/bpy"
	"container/list"
	"sync"
)

type memCacheEnt struct {
	hash    string
	val     []byte
	listEnt *list.Element
}

type MemCachedCStore struct {
	lock    sync.Mutex
	size    uint64
	maxSize uint64
	lru     *list.List
	cache   map[string]*memCacheEnt
	store   bpy.CStore
}

func NewMemCachedCStore(store bpy.CStore, maxSize uint64) bpy.CStore {
	return &MemCachedCStore{
		size:    0,
		maxSize: maxSize,
		lru:     list.New(),
		cache:   make(map[string]*memCacheEnt),
		store:   store,
	}
}

func (m *MemCachedCStore) Get(hash [32]byte) ([]byte, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	cacheEnt, ok := m.cache[string(hash[:])]
	if ok {
		m.lru.MoveToFront(cacheEnt.listEnt)
		return cacheEnt.val, nil
	}
	val, err := m.store.Get(hash)
	if err != nil {
		return nil, err
	}
	for uint64(len(val))+m.size > m.maxSize {
		back := m.lru.Remove(m.lru.Back()).(*memCacheEnt)
		m.size -= uint64(len(back.val))
		delete(m.cache, back.hash)
	}
	newCacheEnt := &memCacheEnt{
		hash: string(hash[:]),
		val:  val,
	}
	listEnt := m.lru.PushFront(newCacheEnt)
	newCacheEnt.listEnt = listEnt
	m.cache[newCacheEnt.hash] = newCacheEnt
	m.size += uint64(len(val))
	return val, nil
}

func (m *MemCachedCStore) Put(val []byte) ([32]byte, error) {
	return m.store.Put(val)
}

func (m *MemCachedCStore) Flush() error {
	return m.store.Flush()
}

func (m *MemCachedCStore) Close() error {
	return m.store.Close()
}
