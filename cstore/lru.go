package cstore

import (
	"acha.ninja/bpy/bpack"
	"container/list"
)

type lruent struct {
	path string
	pack *bpack.Reader
}

type bpacklru struct {
	getfn func(string) (*bpack.Reader, error)
	nents int
	l     *list.List
}

func (lru *bpacklru) get(path string) (*bpack.Reader, error) {
	for e := lru.l.Front(); e != nil; e = e.Next() {
		ent := e.Value.(lruent)
		if ent.path == path {
			lru.l.MoveToFront(e)
			return ent.pack, nil
		}
	}
	pack, err := lru.getfn(path)
	if err != nil {
		return nil, err
	}
	lru.l.PushFront(lruent{path: path, pack: pack})
	if lru.l.Len() > lru.nents {
		ent := lru.l.Remove(lru.l.Back()).(lruent)
		ent.pack.Close()
	}
	return pack, nil
}
