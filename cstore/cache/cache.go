package cache

import (
	"container/list"
	"errors"
	"github.com/boltdb/bolt"
	"os"
	"sync"
)

type Cache struct {
	lock    sync.Mutex
	db      *bolt.DB
	size    uint64
	maxSize uint64
	lruMap  map[[32]byte]*list.Element
	lruList *list.List
}

type cacheEnt struct {
	hash [32]byte
	size uint64
}

func NewCache(dbpath string, mode os.FileMode, maxSize uint64) (*Cache, error) {
	db, err := bolt.Open(dbpath, mode, nil)
	if err != nil {
		return nil, err
	}

	cache := &Cache{
		db:      db,
		lruMap:  make(map[[32]byte]*list.Element),
		lruList: list.New(),
		size:    0,
		maxSize: maxSize,
	}

	err = db.Update(func(tx *bolt.Tx) error {

		b, err := tx.CreateBucketIfNotExists([]byte("cache"))
		if err != nil {
			return err
		}

		err = b.ForEach(func(k, v []byte) error {
			var key [32]byte
			copy(key[:], k)
			e := cache.lruList.PushFront(len(v))
			cache.lruMap[key] = e
			cache.size += uint64(len(v))
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return cache, nil
}

func (c *Cache) Get(hash [32]byte) ([]byte, bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	elem, ok := c.lruMap[hash]
	if !ok {
		return nil, false, nil
	}
	c.lruList.MoveToFront(elem)
	var ret []byte
	err := c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("cache"))
		v := b.Get(hash[:])
		ret = make([]byte, len(v), len(v))
		copy(ret, v)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return ret, true, nil
}

func (c *Cache) Put(hash [32]byte, val []byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ok := c.lruMap[hash]
	if ok {
		return nil
	}

	err := c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("cache"))
		for c.size+uint64(len(val)) > c.maxSize {
			if c.lruList.Len() == 0 {
				return errors.New("value too large for cache")
			}
			elem := c.lruList.Back()
			cacheEnt := elem.Value.(cacheEnt)
			err := b.Delete(cacheEnt.hash[:])
			if err != nil {
				return err
			}
			c.lruList.Remove(elem)
			c.size -= cacheEnt.size
			delete(c.lruMap, cacheEnt.hash)
		}

		err := b.Put(hash[:], val)
		if err != nil {
			return err
		}
		c.size += uint64(len(val))
		elem := c.lruList.PushFront(cacheEnt{hash: hash, size: uint64(len(val))})
		c.lruMap[hash] = elem
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) Close() error {
	return c.db.Close()
}
