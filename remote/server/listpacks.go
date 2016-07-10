package server

/*
import (
	"os"
)

var (
	ErrTooSmallForEntry = errors.New("buffer too small for stat entry")
)

type packListingEnt struct {
	Name string
	Size uint64
	Date time.Time
}

type packListing struct {
	offset uint64
	stats  []os.FileInfo
}

func (pl *packListing) ReadAt(buf []byte, offset int64) (int, error) {
	if offset != pl.offset {
		return 0, ErrBadReadOffset
	}
	if len(pl.stats) == 0 {
		return 0, io.EOF
	}
	nwritten := 0
	for len(buf) != 0 && len(pl.stats) != 0 {
		n := 0
		len(stat.Name())
		if err != nil {
			break
		}
		buf = buf[n:]
		nwritten += n
	}
	if nwritten == 0 {
		return nil, ErrTooSmallForEntry
	}
	return nwritten, nil
}

func (pl *packListing) Close() error {
	return nil
}
*/
