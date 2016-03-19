package testhelp

import (
	"crypto/sha256"
	"errors"
)

type MemStore struct {
	vals map[string][]byte
}

func (m *MemStore) Get(hash [32]byte) ([]byte, error) {
	val, ok := m.vals[string(hash[:])]
	if !ok {
		return nil, errors.New("hash not found in store")
	}
	return val, nil
}

func (m *MemStore) Put(val []byte) ([32]byte, error) {
	hash := sha256.Sum256(val)
	valcpy := make([]byte, len(val), len(val))
	copy(valcpy, val)
	m.vals[string(hash[:])] = valcpy
	return hash, nil
}

func NewMemStore() *MemStore {
	return &MemStore{
		vals: make(map[string][]byte),
	}
}
