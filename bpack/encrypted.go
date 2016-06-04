package bpack

import (
	"acha.ninja/bpy/cryptofile"
	"crypto/aes"
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

func NewEncryptedReader(r io.ReadSeekCloser, key [32]byte) (*Reader, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	r, err = cryptofile.NewReader(r, block)
	if err != nil {
		return nil, err
	}
	return NewReader(r)
}
