package p9

import (
	"acha.ninja/bpy/cmd/bpy/p9/proto9"
	"acha.ninja/bpy/fs"
	"errors"
	"strings"
	"sync"
)

var (
	ErrReadOnly = errors.New("read only")
)

func walk(f File, names []string) (File, []proto9.Qid, error) {
	var werr error
	wqids := make([]proto9.Qid, 0, len(names))

	i := 0
	name := ""
	for i, name = range names {
		if name == "." || name == "" || strings.Index(name, "/") != -1 {
			return nil, nil, ErrBadPath
		}
		if name == ".." {
			parent, err := f.Parent()
			if err != nil {
				return nil, nil, err
			}
			qid, err := parent.Qid()
			if err != nil {
				return nil, nil, err
			}
			f = parent
			wqids = append(wqids, qid)
			continue
		}
		qid, err := f.Qid()
		if err != nil {
			return nil, nil, err
		}
		if !qid.IsDir() {
			werr = ErrNotDir
			goto walkerr
		}
		child, err := f.Child(name)
		if err != nil {
			if err == ErrNotExist {
				werr = ErrNotExist
				goto walkerr
			}
			return nil, nil, err
		}
		f = child
		wqids = append(wqids, qid)
	}
	return f, wqids, nil

walkerr:
	if i == 0 {
		return nil, nil, werr
	}
	return nil, wqids, nil
}

var pathMutex sync.Mutex
var pathCount uint64

func nextPath() uint64 {
	pathMutex.Lock()
	r := pathCount
	pathCount++
	pathMutex.Unlock()
	return r
}

type file struct {
	srv    *Server
	parent *file
	root   [32]byte
	qid    proto9.Qid
	path   string
	dirEnt fs.DirEnt
}

func (f *file) Parent() (File, error) {
	return f.parent, nil
}
func (f *file) Child(name string) (File, error) {
	return nil, errors.New("unimplemented")
}
func (f *file) Qid() (proto9.Qid, error) {
	return f.qid, nil
}
func (f *file) Stat() (proto9.Stat, error) {
	return proto9.Stat{}, errors.New("unimplemented")
}
func (f *file) NewHandle() (Handle, error) {
	return nil, errors.New("unimplemented")
}

type dirHandle struct {
	file   *file
	offset uint64
	stats  []proto9.Stat
}

func (d *dirHandle) GetFile() (File, error) {
	return nil, errors.New("unimplemented")
}
func (d *dirHandle) GetIounit(maxMessageSize uint32) uint32 {
	return 0
}

func (d *dirHandle) Twalk(msg *proto9.Twalk) (File, []proto9.Qid, error) {
	return nil, []proto9.Qid{}, errors.New("unimplemented")
}

func (d *dirHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	return proto9.Qid{}, errors.New("unimplemented")
}

func (d *dirHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	if msg.Offset != d.offset {
		return 0, ErrBadRead
	}
	n := uint32(0)
	for len(d.stats) != 0 {
		curstat := d.stats[0]
		statlen := uint32(proto9.StatLen(&curstat))
		if uint64(statlen+n) > uint64(len(buf)) {
			if n == 0 {
				return 0, proto9.ErrBuffTooSmall
			}
			break
		}
		proto9.PackStat(buf[n:n+statlen], &curstat)
		n += statlen
		d.stats = d.stats[1:]
	}
	d.offset += uint64(n)
	return n, nil
}

func (d *dirHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	return 0, ErrReadOnly
}

func (d *dirHandle) Tcreate(msg *proto9.Tcreate) (Handle, error) {
	return nil, ErrReadOnly
}

func (d *dirHandle) Twstat(msg *proto9.Twstat) error {
	return ErrReadOnly
}

func (d *dirHandle) Tremove(msg *proto9.Tremove) error {
	return ErrReadOnly
}

func (d *dirHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return proto9.Stat{}, errors.New("unimplemented")
}

func (d *dirHandle) Clunk() error {
	return errors.New("unimplemented")
}

type fileHandle struct {
	file *file
	rdr  *fs.FileReader
}

func (f *fileHandle) GetFile() (File, error) {
	return nil, errors.New("unimplemented")
}
func (f *fileHandle) GetIounit(maxMessageSize uint32) uint32 {
	return 0
}

func (f *fileHandle) Twalk(msg *proto9.Twalk) (File, []proto9.Qid, error) {
	return nil, []proto9.Qid{}, errors.New("unimplemented")
}

func (f *fileHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	return proto9.Qid{}, errors.New("unimplemented")
}

func (f *fileHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	return 0, errors.New("unimplemented")
}

func (f *fileHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	return 0, ErrReadOnly
}

func (f *fileHandle) Tcreate(msg *proto9.Tcreate) (Handle, error) {
	return nil, ErrReadOnly
}

func (f *fileHandle) Twstat(msg *proto9.Twstat) error {
	return ErrReadOnly
}

func (f *fileHandle) Tremove(msg *proto9.Tremove) error {
	return ErrReadOnly
}

func (f *fileHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return proto9.Stat{}, errors.New("unimplemented")
}

func (f *fileHandle) Clunk() error {
	return errors.New("unimplemented")
}
