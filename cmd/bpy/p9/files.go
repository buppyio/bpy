package p9

import (
	"errors"
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/p9/proto9"
	"github.com/buppyio/bpy/fs"
	"io"
	"os"
	"path"
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
	path   string
	qid    proto9.Qid
	dirEnt fs.DirEnt
}

func (f *file) Parent() (File, error) {
	return f.parent, nil
}

func (f *file) Child(name string) (File, error) {
	if !f.dirEnt.IsDir() {
		return nil, fmt.Errorf("%s is not a dir", f.path)
	}

	dirEnts, err := fs.ReadDir(f.srv.store, f.dirEnt.Data.Data)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(dirEnts); i++ {
		if dirEnts[i].EntName == name {
			return &file{
				srv:    f.srv,
				parent: f,
				qid:    makeQid(dirEnts[i].IsDir()),
				path:   path.Join(f.path, name),
				dirEnt: dirEnts[i],
			}, nil
		}
	}
	return nil, fmt.Errorf("%s does not exist", path.Join(f.path, name))
}

func (f *file) Qid() (proto9.Qid, error) {
	return f.qid, nil
}

func (f *file) Stat() (proto9.Stat, error) {
	return osToProto9Stat(f.qid, &f.dirEnt), nil
}

func (f *file) NewHandle() (Handle, error) {
	if f.dirEnt.IsDir() {
		return &dirHandle{
			file: f,
		}, nil
	}
	return &fileHandle{
		file: f,
	}, nil
}

type dirHandle struct {
	file   *file
	offset uint64
	stats  []proto9.Stat
}

func (d *dirHandle) GetFile() (File, error) {
	return d.file, nil
}

func (d *dirHandle) GetIounit(maxMessageSize uint32) uint32 {
	return 0
}

func (d *dirHandle) Twalk(msg *proto9.Twalk) (File, []proto9.Qid, error) {
	return walk(d.file, msg.Names)
}

func (d *dirHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	return d.file.qid, nil
}

func osToProto9Stat(qid proto9.Qid, ent os.FileInfo) proto9.Stat {
	mode := proto9.FileMode(ent.Mode() & 0777)
	if ent.Mode().IsDir() {
		mode |= proto9.DMDIR
	}
	return proto9.Stat{
		Mode:   mode,
		Atime:  0,
		Mtime:  0,
		Name:   ent.Name(),
		Qid:    qid,
		Length: uint64(ent.Size()),
		UID:    "nobody",
		GID:    "nobody",
		MUID:   "nobody",
	}
}

func (d *dirHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	if msg.Offset == 0 {
		dirEnts, err := fs.ReadDir(d.file.srv.store, d.file.dirEnt.Data.Data)
		if err != nil {
			return 0, err
		}
		d.stats = make([]proto9.Stat, len(dirEnts), len(dirEnts))
		for i, dirEnt := range dirEnts {
			d.stats[i] = osToProto9Stat(makeQid(dirEnt.IsDir()), &dirEnt)
		}
	}

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
	return d.file.Stat()
}

func (d *dirHandle) Clunk() error {
	return nil
}

type fileHandle struct {
	file *file
	rdr  *fs.FileReader
}

func (f *fileHandle) GetFile() (File, error) {
	return f.file, nil
}
func (f *fileHandle) GetIounit(maxMessageSize uint32) uint32 {
	return 0
}

func (f *fileHandle) Twalk(msg *proto9.Twalk) (File, []proto9.Qid, error) {
	return walk(f.file, msg.Names)
}

func (f *fileHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	if f.rdr != nil {
		f.rdr.Close()
		f.rdr = nil
	}
	var err error
	f.rdr, err = fs.Open(f.file.srv.store, f.file.srv.root, f.file.path)
	return f.file.qid, err
}

func (f *fileHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	if f.rdr == nil {
		return 0, fmt.Errorf("fid for '%s' is not open", f.file.path)
	}
	n, err := f.rdr.Read(buf)
	if n != 0 {
		return uint32(n), nil
	}
	if err == io.EOF {
		return 0, nil
	}
	return 0, err
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
	return f.file.Stat()
}

func (f *fileHandle) Clunk() error {
	if f.rdr != nil {
		f.rdr.Close()
		f.rdr = nil
	}
	return nil
}
