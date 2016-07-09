package client

import (
	"acha.ninja/bpy/remote/proto"
	"errors"
)

type File struct {
	c      *Client
	fid    uint32
	offset uint64
}

func (f *File) Read(buf []byte) (int, error) {
	maxn := f.c.maxMessageSize - proto.READOVERHEAD
	n := uint32(len(buf))
	if n > maxn {
		n = maxn
	}
	resp, err := f.c.TReadAt(f.fid, f.offset, n)
	if err != nil {
		return 0, err
	}
	return copy(buf, resp.Data), nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		f.offset += uint64(offset)
		return int64(f.offset), nil
	default:
		return int64(f.offset), errors.New("seek unsupported")
	}
}

func (f *File) Close() error {
	_, err := f.c.TClose(f.fid)
	return err
}
