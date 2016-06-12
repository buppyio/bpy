package bpy

import (
	"encoding/hex"
	"errors"
)

type CStoreReader interface {
	Get([32]byte) ([]byte, error)
	Close() error
}

type CStoreWriter interface {
	Put([]byte) ([32]byte, error)
	Close() error
}

type Key struct {
	CipherKey [32]byte
	HmacKey   [32]byte
	Id        [32]byte
}

func ParseHash(hashstr string) ([32]byte, error) {
	var hash [32]byte
	if len(hashstr) != 64 {
		return hash, errors.New("bad length")
	}
	_, err := hex.Decode(hash[:], []byte(hashstr))
	if err != nil {
		return hash, err
	}
	return hash, nil
}
