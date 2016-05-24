package remote

import (
	"acha.ninja/bpy/proto9"
	"acha.ninja/bpy/server9"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"regexp"
	"strings"
	// "log"
)

const (
	TAGMAXSIZE  = 2048
	TAGREGEXSTR = "[a-zA-Z0-9_]+"
)

var tagRegex = regexp.MustCompile(TAGREGEXSTR)

type CtlFile struct {
	dbPath string
}

func (f *CtlFile) Remove() error {
	return ErrReadOnly
}

func (f *CtlFile) Parent() (server9.File, error) {
	return nil, server9.ErrBadPath
}

func (f *CtlFile) Child(name string) (server9.File, error) {
	return nil, server9.ErrBadPath
}

func (f *CtlFile) NewHandle() (server9.Handle, error) {
	return &CtlFileHandle{
		file: f,
	}, nil
}

func (f *CtlFile) Stat() (proto9.Stat, error) {
	return proto9.Stat{
		Mode:   0644,
		Atime:  0,
		Mtime:  0,
		Name:   CTLFILENAME,
		Qid:    makeQid(false),
		Length: 0,
		UID:    "nobody",
		GID:    "nobody",
		MUID:   "nobody",
	}, nil
}

func (f *CtlFile) Qid() (proto9.Qid, error) {
	stat, err := f.Stat()
	if err != nil {
		return proto9.Qid{}, err
	}
	return stat.Qid, nil
}

type CtlFileHandle struct {
	file *CtlFile
	db   *bolt.DB
}

func (fh *CtlFileHandle) GetFile() (server9.File, error) {
	return fh.file, nil
}

func (fh *CtlFileHandle) GetIounit(maxMessageSize uint32) uint32 {
	return maxMessageSize - proto9.WriteOverhead
}

func (fh *CtlFileHandle) Tcreate(msg *proto9.Tcreate) (server9.Handle, error) {
	return nil, server9.ErrNotDir
}

func (fh *CtlFileHandle) Tremove(msg *proto9.Tremove) error {
	return fh.Clunk()
}

func (fh *CtlFileHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	return nil, nil, server9.ErrNotDir
}

func (fh *CtlFileHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return fh.file.Stat()
}

func (fh *CtlFileHandle) Twstat(msg *proto9.Twstat) error {
	return ErrReadOnly
}

func (fh *CtlFileHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	if fh.db != nil {
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
	fh.db, err = openTagDB(fh.file.dbPath)
	if err != nil {
		return proto9.Qid{}, err
	}
	return qid, nil
}

func (fh *CtlFileHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	return 0, errors.New("ctl file is write only")
}

func (fh *CtlFileHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	cmd := string(msg.Data)
	err := ctlCommand(fh.db, cmd)
	if err != nil {
		return 0, err
	}
	return uint32(len(msg.Data)), nil
}

func (fh *CtlFileHandle) Clunk() error {
	if fh.db != nil {
		fh.db.Close()
		fh.db = nil
	}
	return nil
}

func ctlCommand(db *bolt.DB, cmd string) error {
	args := strings.Split(cmd, " ")
	if len(args) < 1 {
		return errors.New("not enough arguments to ctl command")
	}
	switch args[0] {
	case "set":
		if len(args) != 3 {
			return errors.New("ctl set requires 2 arguments")
		}
		return ctlSetTag(db, args[1], args[2])
	case "cas":
		if len(args) != 4 {
			return errors.New("ctl cas requires 3 arguments")
		}
		return ctlCasTag(db, args[1], args[2], args[3])
	}
	return fmt.Errorf("invalid ctl command: '%s'", args[0])
}

func validateTagPair(tag, value string) error {
	if len(tag) > TAGMAXSIZE {
		return errors.New("tag key too large")
	}
	if len(value) > TAGMAXSIZE {
		return errors.New("tag val too large")
	}
	if !tagRegex.MatchString(tag) {
		return errors.New("tag key invalid it must match: " + TAGREGEXSTR)
	}
	return nil
}

func ctlCasTag(db *bolt.DB, tag, oldval, newval string) error {
	err := validateTagPair(tag, newval)
	if err != nil {
		return err
	}
	stale := false
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tags"))
		if string(b.Get([]byte(tag))) != oldval {
			stale = true
			return nil
		}
		return b.Put([]byte(tag), []byte(newval))
	})
	if err != nil {
		return nil
	}
	if stale {
		return errors.New("tag update failed, value stale due to concurrent modification")
	}
	return nil
}

func ctlSetTag(db *bolt.DB, tag, value string) error {
	err := validateTagPair(tag, value)
	if err != nil {
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tags"))
		return b.Put([]byte(tag), []byte(value))
	})
	if err != nil {
		return nil
	}
	return nil
}
