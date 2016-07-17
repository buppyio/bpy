package server

import (
	"encoding/binary"
	"io"
)

type tagListingFile struct {
	offset  uint64
	tagFile string
	entries []string
}

func listTags(dir string) ([]string, error) {
	listing := make([]string, 0, 32)
	return listing, nil
}

func (tl *tagListingFile) ReadAtOffset(buf []byte, offset uint64) (int, error) {
	if offset == 0 {
		listing, err := listTags(tl.tagFile)
		if err != nil {
			return 0, err
		}
		tl.offset = 0
		tl.entries = listing
	}
	if offset != tl.offset {
		return 0, ErrBadReadOffset
	}
	if len(tl.entries) == 0 {
		return 0, io.EOF
	}
	nwritten := 0
	for len(buf) != 0 && len(tl.entries) != 0 {
		ent := tl.entries[0]
		n := 2 + len(ent)
		if len(buf) < n {
			break
		}
		binary.BigEndian.PutUint16(buf[0:2], uint16(len(ent)))
		copy(buf[2:2+len(ent)], []byte(ent))
		buf = buf[n:]
		tl.entries = tl.entries[1:]
		nwritten += n
	}
	if nwritten == 0 {
		return 0, ErrTooSmallForEntry
	}
	tl.offset += uint64(nwritten)
	return nwritten, nil
}

func (pl *tagListingFile) Close() error {
	return nil
}
