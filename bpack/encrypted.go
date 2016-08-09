package bpack

import (
	"crypto/aes"
	"github.com/buppyio/bpy/cryptofile"
	"io"
)

func NewEncryptedWriter(w io.WriteCloser, key [32]byte) (*Writer, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	w, err = cryptofile.NewWriter(w, block)
	if err != nil {
		return nil, err
	}
	return NewWriter(w)
}

func NewEncryptedReader(r ReadSeekCloser, key [32]byte, fsize int64) (*Reader, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	cryptof, err := cryptofile.NewReader(r, block, fsize)
	if err != nil {
		return nil, err
	}
	dataLen, err := cryptof.Size()
	if err != nil {
		return nil, err
	}
	return NewReader(cryptof, uint64(dataLen)), nil
}
