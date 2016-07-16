package cryptofile

import (
	"crypto/cipher"
	"errors"
	"io"
)

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type Reader struct {
	r      ReadSeekCloser
	block  cipher.Block
	size   int64
	offset int64
	ctr    *ctrState
	rbuf   [4096]byte
}

func NewReader(r ReadSeekCloser, block cipher.Block, fsize int64) (*Reader, error) {
	if fsize%int64(block.BlockSize()) != 0 {
		return nil, errors.New("file size is not a multiple of block size")
	}
	iv := make([]byte, block.BlockSize(), block.BlockSize())
	_, err := r.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	_, err = io.ReadFull(r, iv)
	if err != nil {
		return nil, err
	}
	return &Reader{
		r:     r,
		block: block,
		size:  fsize - int64(len(iv)),
		ctr:   newCtrState(iv),
	}, nil
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	if whence != io.SeekStart {
		return r.offset, errors.New("unsupported whence")
	}
	if offset < 0 {
		offset = 0
	}
	if offset > r.size {
		offset = r.size
	}
	r.offset = offset
	return r.offset, nil
}

func (r *Reader) Size() (int64, error) {
	buf := make([]byte, r.block.BlockSize(), r.block.BlockSize())
	n, err := r.readBlocks((r.size/int64(r.block.BlockSize()))-1, buf)
	if err != nil && err != io.EOF {
		return 0, err
	}
	return r.size - int64(r.block.BlockSize()) + int64(n), nil
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

	_, err := r.r.Seek((1+idx)*blocksz, io.SeekStart)
	if err != nil {
		return 0, err
	}

	_, err = io.ReadFull(r.r, buf)
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

func (r *Reader) Close() error {
	return r.r.Close()
}
