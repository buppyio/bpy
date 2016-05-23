package remote

import (
	"acha.ninja/bpy/proto9"
	"acha.ninja/bpy/server9"
	// "log"
)

type CtlFile struct {
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
	f, err := fh.GetFile()
	if err != nil {
		return proto9.Qid{}, err
	}
	qid, err := f.Qid()
	if err != nil {
		return proto9.Qid{}, err
	}
	return qid, nil
}

func (fh *CtlFileHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	return 0, nil
}

func (fh *CtlFileHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	return uint32(len(msg.Data)), nil
}

func (fh *CtlFileHandle) Clunk() error {
	return nil
}
