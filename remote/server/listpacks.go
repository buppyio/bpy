package server

import (
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"time"
)

var (
	ErrTooSmallForEntry = errors.New("buffer too small for stat entry")
	ErrBadReadOffset    = errors.New("bad read offset")
)

type packListingEnt struct {
	Name string
	Size uint64
	Date time.Time
}

type packListing struct {
	offset  uint64
	packDir string
	entries []packListingEnt
}

func listPacks(dir string) ([]packListingEnt, error) {
	listing := make([]packListingEnt, 0, 32)
	stats, err := ioutil.ReadDir(dir)
	if err != nil {
		return listing, err
	}
	for _, stat := range stats {
		if !strings.HasSuffix(stat.Name(), ".tmp") {
			listing = append(listing, packListingEnt{
				Name: stat.Name(),
				Size: uint64(stat.Size()),
				Date: stat.ModTime(),
			})
		}
	}
	return listing, nil
}

func (pl *packListing) ReadAtOffset(buf []byte, offset uint64) (int, error) {
	if offset == 0 {
		listing, err := listPacks(pl.packDir)
		if err != nil {
			return 0, err
		}
		pl.offset = 0
		pl.entries = listing
	}
	if offset != pl.offset {
		return 0, ErrBadReadOffset
	}
	if len(pl.entries) == 0 {
		return 0, io.EOF
	}
	nwritten := 0
	for len(buf) != 0 && len(pl.entries) != 0 {
		ent := &pl.entries[0]
		n := 2 + len(ent.Name) + 8 + 8
		if len(buf) < n {
			break
		}
		binary.BigEndian.PutUint16(buf[0:2], uint16(len(ent.Name)))
		copy(buf[2:2+len(ent.Name)], []byte(ent.Name))
		binary.BigEndian.PutUint64(buf[2+len(ent.Name):10+len(ent.Name)], uint64(ent.Size))
		binary.BigEndian.PutUint64(buf[10+len(ent.Name):18+len(ent.Name)], uint64(ent.Date.Unix()))
		buf = buf[n:]
		pl.entries = pl.entries[1:]
		nwritten += n
	}
	if nwritten == 0 {
		return 0, ErrTooSmallForEntry
	}
	pl.offset += uint64(nwritten)
	return nwritten, nil
}

func (pl *packListing) Close() error {
	return nil
}
