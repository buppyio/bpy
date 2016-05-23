package remote

import (
	"acha.ninja/bpy/proto9"
	"acha.ninja/bpy/server9"
	"bytes"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"io"
	"time"
)

type TagFile struct {
	dbpath string
	name   string
	parent server9.File
}

func (f *TagFile) Remove() error {
	return errors.New("unimplemented remove tag")
}

func (f *TagFile) Parent() (server9.File, error) {
	return f.parent, nil
}

func (f *TagFile) Child(name string) (server9.File, error) {
	return nil, server9.ErrNotDir
}

func (f *TagFile) NewHandle() (server9.Handle, error) {
	return &TagFileHandle{
		file: f,
	}, nil
}

func (f *TagFile) Stat() (proto9.Stat, error) {
	return proto9.Stat{
		Mode:   0644,
		Atime:  0,
		Mtime:  0,
		Name:   f.name,
		Qid:    makeQid(false),
		Length: 0, // XXX: does this matter?
		UID:    "nobody",
		GID:    "nobody",
		MUID:   "nobody",
	}, nil
}

func (f *TagFile) Qid() (proto9.Qid, error) {
	stat, err := f.Stat()
	if err != nil {
		return proto9.Qid{}, err
	}
	return stat.Qid, nil
}

type TagFileHandle struct {
	file     *TagFile
	contents []byte
	rdr      *bytes.Reader
}

func (fh *TagFileHandle) GetFile() (server9.File, error) {
	return fh.file, nil
}

func (fh *TagFileHandle) GetIounit(maxMessageSize uint32) uint32 {
	return maxMessageSize - proto9.WriteOverhead
}

func (fh *TagFileHandle) Tcreate(msg *proto9.Tcreate) (server9.Handle, error) {
	return nil, server9.ErrNotDir
}

func (fh *TagFileHandle) Tremove(msg *proto9.Tremove) error {
	fh.file.Remove()
	return fh.Clunk()
}

func (fh *TagFileHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	return nil, nil, server9.ErrNotDir
}

func (fh *TagFileHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return fh.file.Stat()
}

func (fh *TagFileHandle) Twstat(msg *proto9.Twstat) error {
	return ErrReadOnly
}

func (fh *TagFileHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	if fh.contents != nil {
		return proto9.Qid{}, server9.ErrFileAlreadyOpen
	}
	f, err := fh.GetFile()
	if err != nil {
		return proto9.Qid{}, err
	}
	qid, err := f.Qid()
	if err != nil {
		return proto9.Qid{}, err
	}
	db, err := openTagDB(fh.file.dbpath)
	if err != nil {
		return proto9.Qid{}, err
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tags"))
		buf := b.Get([]byte(fh.file.name))
		fh.contents = make([]byte, len(buf), len(buf))
		copy(fh.contents, buf)
		fh.rdr = bytes.NewReader(fh.contents)
		return nil
	})
	return qid, nil
}

func (fh *TagFileHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	if fh.contents == nil {
		return 0, server9.ErrFileNotOpen
	}
	n, err := fh.rdr.ReadAt(buf, int64(msg.Offset))
	if n != 0 {
		return uint32(n), nil
	}
	if err == io.EOF {
		return 0, nil
	}
	return 0, err
}

func (fh *TagFileHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	return 0, ErrReadOnly
}

func (fh *TagFileHandle) Clunk() error {
	return nil
}

type TagDirFile struct {
	dbpath string
	parent server9.File
}

func (f *TagDirFile) Remove() error {
	return ErrReadOnly
}

func (f *TagDirFile) Parent() (server9.File, error) {
	return f.parent, nil
}

func (f *TagDirFile) Child(name string) (server9.File, error) {
	db, err := openTagDB(f.dbpath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	found := true
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tags"))
		val := b.Get([]byte(name))
		if val == nil {
			found = false
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, server9.ErrNotExist
	}
	return &TagFile{
		dbpath: f.dbpath,
		parent: f,
		name:   name,
	}, nil
}

func (f *TagDirFile) NewHandle() (server9.Handle, error) {
	return &TagDirHandle{
		file: f,
	}, nil
}

func (f *TagDirFile) Stat() (proto9.Stat, error) {
	return proto9.Stat{
		Mode:   0644,
		Atime:  0,
		Mtime:  0,
		Name:   TAGDIRNAME,
		Qid:    makeQid(true),
		Length: 0,
		UID:    "nobody",
		GID:    "nobody",
		MUID:   "nobody",
	}, nil
}

func (f *TagDirFile) Qid() (proto9.Qid, error) {
	stat, err := f.Stat()
	if err != nil {
		return proto9.Qid{}, err
	}
	return stat.Qid, nil
}

type TagDirHandle struct {
	file  *TagDirFile
	stats server9.StatList
}

func (dh *TagDirHandle) GetFile() (server9.File, error) {
	return dh.file, nil
}

func (dh *TagDirHandle) GetIounit(maxMessageSize uint32) uint32 {
	return maxMessageSize - proto9.WriteOverhead
}

func (dh *TagDirHandle) Tcreate(msg *proto9.Tcreate) (server9.Handle, error) {
	return nil, ErrReadOnly
}

func (dh *TagDirHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	return server9.Walk(dh.file, msg.Names)
}

func (dh *TagDirHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return dh.file.Stat()
}

func (dh *TagDirHandle) Twstat(msg *proto9.Twstat) error {
	return ErrReadOnly
}

func (dh *TagDirHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	return dh.file.Qid()
}

func (dh *TagDirHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	if msg.Offset == 0 {
		db, err := openTagDB(dh.file.dbpath)
		if err != nil {
			return 0, err
		}
		defer db.Close()
		stats := make([]proto9.Stat, 0, 64)
		err = db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("tags"))
			c := b.Cursor()
			for k, _ := c.First(); k != nil; k, _ = c.Next() {
				stats = append(stats, proto9.Stat{
					Mode:   0644,
					Atime:  0,
					Mtime:  0,
					Name:   string(k),
					Qid:    makeQid(true),
					Length: 0,
					UID:    "nobody",
					GID:    "nobody",
					MUID:   "nobody",
				})
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
		dh.stats = server9.StatList{
			Stats: stats,
		}
	}
	return dh.stats.Tread(msg, buf)
}

func (dh *TagDirHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	return 0, server9.ErrBadWrite
}

func (dh *TagDirHandle) Tremove(msg *proto9.Tremove) error {
	return dh.Clunk()
}

func (dh *TagDirHandle) Clunk() error {
	dh.stats = server9.StatList{}
	return nil
}

func openTagDB(dbpath string) (*bolt.DB, error) {
	db, err := bolt.Open(dbpath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("tags"))
		if err != nil {
			return fmt.Errorf("create db bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
