package remote

import (
	"acha.ninja/bpy/proto9"
	"acha.ninja/bpy/server9"
	"errors"
)

const (
	CTLFILENAME = "ctl"
	TAGDIRNAME  = "tags"
	PACKDIRNAME = "packs"
)

var (
	ErrReadOnly = errors.New("read only")
)

type RootFile struct {
	packDir server9.File
	ctlFile server9.File
	tagDir  server9.File
}

func (f *RootFile) Wstat(stat proto9.Stat) error {
	return ErrReadOnly
}

func (f *RootFile) Remove() error {
	return ErrReadOnly
}

func (f *RootFile) Parent() (server9.File, error) {
	return nil, server9.ErrBadPath
}

func (f *RootFile) Child(name string) (server9.File, error) {
	switch name {
	case "ctl":
		return f.ctlFile, nil
	case "packs":
		return f.packDir, nil
	case "tags":
		return f.tagDir, nil
	}
	return nil, server9.ErrNotExist
}

func (f *RootFile) NewHandle() (server9.Handle, error) {
	return &RootHandle{
		file: f,
	}, nil
}

func (f *RootFile) Stat() (proto9.Stat, error) {
	return proto9.Stat{
		Mode:   0644,
		Atime:  0,
		Mtime:  0,
		Name:   "",
		Qid:    makeQid(true),
		Length: 0,
		UID:    "nobody",
		GID:    "nobody",
		MUID:   "nobody",
	}, nil
}

func (f *RootFile) Qid() (proto9.Qid, error) {
	return makeQid(true), nil
}

type RootHandle struct {
	file  *RootFile
	stats server9.StatList
}

func (rh *RootHandle) GetFile() (server9.File, error) {
	return rh.file, nil
}

func (rh *RootHandle) GetIounit(maxMessageSize uint32) uint32 {
	return maxMessageSize - proto9.WriteOverhead
}

func (rh *RootHandle) Tcreate(msg *proto9.Tcreate) (server9.Handle, error) {
	return nil, ErrReadOnly
}

func (rh *RootHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	return server9.Walk(rh.file, msg.Names)
}

func (rh *RootHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return rh.file.Stat()
}

func (rh *RootHandle) Twstat(msg *proto9.Twstat) error {
	return ErrReadOnly
}

func (rh *RootHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	return makeQid(true), nil
}

func (rh *RootHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	if msg.Offset == 0 {

		ctlStat, err := rh.file.ctlFile.Stat()
		if err != nil {
			return 0, nil
		}

		packStat, err := rh.file.packDir.Stat()
		if err != nil {
			return 0, nil
		}

		tagStat, err := rh.file.tagDir.Stat()
		if err != nil {
			return 0, nil
		}

		stats := []proto9.Stat{
			ctlStat,
			packStat,
			tagStat,
		}
		rh.stats = server9.StatList{
			Stats: stats,
		}
	}
	return rh.stats.Tread(msg, buf)
}

func (rh *RootHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	return 0, server9.ErrBadWrite
}

func (rh *RootHandle) Tremove(msg *proto9.Tremove) error {
	return ErrReadOnly
}

func (rh *RootHandle) Clunk() error {
	return nil
}
