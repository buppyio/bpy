package cache

import (
	"errors"
	"github.com/boltdb/bolt"
	"os"
	"sync"
)

type lruListEnt struct {
	Next, Prev *lruListEnt
	Hash       [32]byte
	Size       uint64
}

type lruList struct {
	root lruListEnt
	Len  int64
}

func (l *lruList) Init() {
	l.root.Next = &l.root
	l.root.Prev = &l.root
	l.Len = 0
}

func (l *lruList) Front() *lruListEnt {
	if l.Len == 0 {
		return nil
	}
	return l.root.Next
}

func (l *lruList) Back() *lruListEnt {
	if l.Len == 0 {
		return nil
	}
	return l.root.Prev
}

func (l *lruList) PushFront(hash [32]byte, size uint64) *lruListEnt {
	toAdd := &lruListEnt{
		Hash: hash,
		Size: size,
	}
	n := l.root.Next
	l.root.Next = toAdd
	toAdd.Prev = &l.root
	toAdd.Next = n
	n.Prev = toAdd
	l.Len++
	return toAdd
}

func (l *lruList) Remove(toRemove *lruListEnt) {
	toRemove.Next.Prev = toRemove.Prev
	toRemove.Prev.Next = toRemove.Next
	toRemove.Next = nil
	toRemove.Prev = nil
	l.Len--
}

func (l *lruList) MoveToFront(toMove *lruListEnt) {
	l.Remove(toMove)
	n := l.root.Next
	l.root.Next = toMove
	toMove.Prev = &l.root
	toMove.Next = n
	n.Prev = toMove
	l.Len++
}

type Cache struct {
	lock    sync.Mutex
	db      *bolt.DB
	size    uint64
	maxSize uint64
	lruMap  map[[32]byte]*lruListEnt
	lruList lruList

	pendingPut map[[32]byte][]byte
	pendingDel map[[32]byte]struct{}
}

func NewCache(dbpath string, mode os.FileMode, maxSize uint64) (*Cache, error) {
	db, err := bolt.Open(dbpath, mode, nil)
	if err != nil {
		return nil, err
	}

	cache := &Cache{
		db:         db,
		lruMap:     make(map[[32]byte]*lruListEnt),
		size:       0,
		maxSize:    maxSize,
		pendingPut: make(map[[32]byte][]byte),
		pendingDel: make(map[[32]byte]struct{}),
	}
	cache.lruList.Init()

	err = db.Update(func(tx *bolt.Tx) error {

		b, err := tx.CreateBucketIfNotExists([]byte("cache"))
		if err != nil {
			return err
		}

		err = b.ForEach(func(k, v []byte) error {
			var key [32]byte
			copy(key[:], k)
			e := cache.lruList.PushFront(key, uint64(len(v)))
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

	ret, ok := c.pendingPut[hash]
	if ok {
		return ret, true, nil
	}

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

	for c.size+uint64(len(val)) > c.maxSize {
		if c.lruList.Len == 0 {
			return errors.New("value too large for cache")
		}
		elem := c.lruList.Back()
		c.lruList.Remove(elem)
		c.size -= elem.Size
		delete(c.lruMap, elem.Hash)
		delete(c.pendingPut, elem.Hash)
		c.pendingDel[elem.Hash] = struct{}{}

	}

	c.size += uint64(len(val))
	elem := c.lruList.PushFront(hash, uint64(len(val)))
	c.lruMap[hash] = elem
	c.pendingPut[hash] = val
	delete(c.pendingDel, hash)

	if len(c.pendingDel) > 1000 || len(c.pendingPut) > 1000 {
		err := c.flushPending()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cache) flushPending() error {

	err := c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("cache"))

		for k, v := range c.pendingPut {
			err := b.Put(k[:], v)
			if err != nil {
				return err
			}
		}

		for k, _ := range c.pendingDel {
			err := b.Delete(k[:])
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	c.pendingPut = make(map[[32]byte][]byte)
	c.pendingDel = make(map[[32]byte]struct{})
	return nil
}

func (c *Cache) Close() error {
	err := c.flushPending()
	if err != nil {
		return err
	}
	return c.db.Close()
}
