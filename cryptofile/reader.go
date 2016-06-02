package cryptofile

import (
	"crypto/cipher"
	"io"
)

type Reader struct {
	r      io.ReaderAt
	block  cipher.Block
	size   int64
	offset int64
	ctr    *ctrState
	rbuf   [4096]byte
}

func NewReader(r io.ReaderAt, block cipher.Block, iv []byte, size int64) *Reader {
	return &Reader{
		r:     r,
		block: block,
		size:  size,
		ctr:   newCtrState(iv),
	}
}

func (r *Reader) readBlocks(idx int64, buf []byte) (int, error) {

	blocksz := int64(r.block.BlockSize())

	if int64(len(buf))%blocksz != 0 {
		panic("bufsize not multiple of blocksize")
	}
	nblocks := int64(len(buf)) / blocksz

	if idx*blocksz >= r.size {
		return 0, io.EOF
	}

	if r.size < idx*blocksz+int64(len(buf)) {
		buf = buf[:r.size-idx*blocksz]
	}

	_, err := r.r.ReadAt(buf, idx*blocksz)
	if err != nil {
		return 0, err
	}

	r.ctr.Reset()
	r.ctr.Add(uint64(idx))
	for i := int64(0); i < nblocks; i++ {
		block := buf[i*blocksz : (i+1)*blocksz]
		r.block.Decrypt(block, block)
		r.ctr.Xor(block)
		r.ctr.Add(1)
	}

	if idx*blocksz+int64(len(buf)) == r.size {
		for i := len(buf) - 1; ; i-- {
			if buf[i] == 0x80 {
				buf = buf[:i]
				break
			}
		}
	}

	if len(buf) == 0 {
		return 0, io.EOF
	}

	return len(buf), nil
}

func (r *Reader) Read(buf []byte) (int, error) {

	if len(buf) == 0 {
		return 0, nil
	}

	aligned := r.offset
	if aligned%int64(r.block.BlockSize()) != 0 {
		aligned -= (aligned % int64(r.block.BlockSize()))
	}
	shiftamnt := r.offset - aligned

	buflen := int64(len(buf)) + r.offset - aligned
	if buflen%int64(r.block.BlockSize()) != 0 {
		buflen += int64(r.block.BlockSize()) - (buflen % int64(r.block.BlockSize()))
	}

	var buf2 []byte
	if buflen < int64(len(r.rbuf)) {
		buf2 = r.rbuf[:buflen]
	} else {
		buf2 = make([]byte, buflen, buflen)
	}

	startidx := aligned / int64(r.block.BlockSize())

	nread, err := r.readBlocks(startidx, buf2)
	n := copy(buf, buf2[shiftamnt:nread])
	r.offset += int64(n)
	if n == 0 {
		if err == nil {
			return 0, io.EOF
		}
	}
	return n, err
}
