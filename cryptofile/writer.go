package foo

import (
	"crypto/cipher"
	"io"
)

type Writer struct {
	w     io.Writer
	block cipher.Block
	buf   []byte
	nbuf  int
}

func NewWriter(b cipher.Block, w io.Writer) *Writer {
	return &Writer{
		w:     w,
		block: b,
		buf:   make([]byte, b.BlockSize()),
	}
}

func (w *Writer) flushBlock() error {
	w.nbuf = 0
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
