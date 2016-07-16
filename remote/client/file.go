package client

import (
	"acha.ninja/bpy/remote/proto"
	"errors"
	"io"
)

type File struct {
	c      *Client
	fid    uint32
	offset uint64
}

func (f *File) Read(buf []byte) (int, error) {
	maxn := f.c.getMaxMessageSize() - proto.READOVERHEAD
	n := uint32(len(buf))
	if n > maxn {
		n = maxn
	}
	resp, err := f.c.TReadAt(f.fid, f.offset, n)
	if err != nil {
		return 0, err
	}
	if len(resp.Data) == 0 {
		return 0, io.EOF
	}
	ncopied := copy(buf, resp.Data)
	f.offset += uint64(ncopied)
	return ncopied, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.offset = uint64(offset)
		return int64(f.offset), nil
	default:
		return int64(f.offset), errors.New("seek unsupported")
	}
}

func (f *File) Close() error {
	f.c.freeFid(f.fid)
	_, err := f.c.TClose(f.fid)
	return err
}
