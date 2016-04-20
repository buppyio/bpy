package bpy

import (
	"encoding/hex"
	"errors"
	"fmt"
)

type CStoreReader interface {
	Get([32]byte) ([]byte, error)
	Close() error
}

type CStoreWriter interface {
	Put([]byte) ([32]byte, error)
	Close() error
}

func ParseHash(hashstr string) ([32]byte, error) {
	var hash [32]byte
	if len(hashstr) != 64 {
		return hash, errors.New("cannot parse hash: bad length")
	}
	_, err := hex.Decode(hash[:], []byte(hashstr))
	if err != nil {
		return hash, fmt.Errorf("cannot parse hash: %s", err.Error())
	}
	return hash, nil
}
