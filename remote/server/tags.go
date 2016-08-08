package server

import (
	"encoding/binary"
	"encoding/json"
	"github.com/boltdb/bolt"
	"io"
	"time"
)

type tagListingFile struct {
	offset    uint64
	tagDBPath string
	entries   []string
}

type gcState struct {
	Generation uint64
	ID         string
}

func getGCState(tx *bolt.Tx) (gcState, error) {
	stateBucket := tx.Bucket([]byte(GCStateBucketName))
	stateBytes := stateBucket.Get([]byte("state"))
	state := gcState{}
	err := json.Unmarshal(stateBytes, &state)
	if err != nil {
		return state, err
	}
	return state, err
}

func setGCState(tx *bolt.Tx, state gcState) error {
	stateBucket := tx.Bucket([]byte(GCStateBucketName))
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return stateBucket.Put([]byte("state"), data)
}

func openTagDB(dbPath string) (*bolt.DB, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(TagBucketName))
		if err != nil {
			return err
		}
		gcStateBucket, err := tx.CreateBucketIfNotExists([]byte(GCStateBucketName))
		if err != nil {
			return err
		}
		if gcStateBucket.Get([]byte("state")) == nil {
			err = setGCState(tx, gcState{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func listTags(dbPath string) ([]string, error) {
	listing := make([]string, 0, 32)
	db, err := openTagDB(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tags"))
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			listing = append(listing, string(k))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return listing, nil
}

func (tl *tagListingFile) ReadAtOffset(buf []byte, offset uint64) (int, error) {
	if offset == 0 {
		listing, err := listTags(tl.tagDBPath)
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
