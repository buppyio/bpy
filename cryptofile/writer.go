package cryptofile

import (
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type Writer struct {
	w     io.Writer
	block cipher.Block
	buf   []byte
	nbuf  int
	ctr   *ctrState
}

func NewWriter(w io.Writer, b cipher.Block) (*Writer, error) {
	iv := make([]byte, b.BlockSize(), b.BlockSize())
	_, err := io.ReadFull(rand.Reader, iv)
	if err != nil {
		return nil, err
	}
	_, err = w.Write(iv)
	if err != nil {
		return nil, err
	}
	return &Writer{
		w:     w,
		block: b,
		buf:   make([]byte, b.BlockSize()),
		ctr:   newCtrState(iv),
	}, nil
}

func (w *Writer) flushBlock() error {
	w.nbuf = 0
	w.ctr.Xor(w.buf)
	w.ctr.Add(1)
	w.block.Encrypt(w.buf, w.buf)
	_, err := w.w.Write(w.buf)
	return err
}

func (w *Writer) Write(buf []byte) (int, error) {
	n := copy(w.buf[w.nbuf:], buf)
	w.nbuf += n
	if w.nbuf != len(w.buf) {
		return n, nil
	}
	err := w.flushBlock()
	if err != nil {
		return n, err
	}
	nw, err := w.Write(buf[n:])
	return n + nw, err
}

func (w *Writer) Close() error {
	for i := w.nbuf; i < len(w.buf); i++ {
		w.buf[i] = 0
	}
	w.buf[w.nbuf] = 0x80
	return w.flushBlock()
}
