package remote

import (
	"acha.ninja/bpy/proto9"
	"acha.ninja/bpy/server9"
	"io"
	"io/ioutil"
	// "log"
	"os"
	"path/filepath"
)

type File struct {
	path   string
	parent server9.File
}

func (f *File) Wstat(stat proto9.Stat) error {
	oldpath := f.path
	dir := filepath.Dir(oldpath)
	newpath := filepath.Join(dir, stat.Name)
	err := os.Rename(oldpath, newpath)
	if err != nil {
		return err
	}
	f.path = newpath
	return nil
}

func (f *File) Remove() error {
	return os.Remove(f.path)
}

func (f *File) Parent() (server9.File, error) {
	if f.parent == nil {
		return nil, server9.ErrBadPath
	}
	return f.parent, nil
}

func (f *File) Child(name string) (server9.File, error) {
	dirents, err := ioutil.ReadDir(f.path)
	if err != nil {
		return nil, err
	}
	for _, ent := range dirents {
		if ent.Name() == name {
			return &File{
				path:   filepath.Join(f.path, name),
				parent: f,
			}, nil
		}
	}
	return nil, server9.ErrNotExist
}

func (f *File) NewHandle() (server9.Handle, error) {
	stat, err := os.Stat(f.path)
	if err != nil {
		return nil, err
	}
	if stat.Mode().IsDir() {
		return &DirHandle{
			file: f,
		}, nil
	}
	return &FileHandle{
		file: f,
	}, nil
}

func (f *File) Stat() (proto9.Stat, error) {
	stat, err := os.Stat(f.path)
	if err != nil {
		return proto9.Stat{}, err
	}
	return osToStat(stat), nil
}

func (f *File) Qid() (proto9.Qid, error) {
	stat, err := os.Stat(f.path)
	if err != nil {
		return proto9.Qid{}, err
	}
	return makeQid(stat.Mode().IsDir()), nil
}

type FileHandle struct {
	file   *File
	osfile *os.File
}

func (fh *FileHandle) GetFile() (server9.File, error) {
	return fh.file, nil
}

func (fh *FileHandle) GetIounit(maxMessageSize uint32) uint32 {
	return maxMessageSize - proto9.WriteOverhead
}

func (fh *FileHandle) Tcreate(msg *proto9.Tcreate) (server9.Handle, error) {
	return nil, server9.ErrNotDir
}

func (fh *FileHandle) Tremove(msg *proto9.Tremove) error {
	fh.file.Remove()
	return fh.Clunk()
}

func (fh *FileHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	return nil, nil, server9.ErrNotDir
}

func (fh *FileHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return fh.file.Stat()
}

func (fh *FileHandle) Twstat(msg *proto9.Twstat) error {
	return fh.file.Wstat(msg.Stat)
}

func (fh *FileHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	if fh.osfile != nil {
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
	fh.osfile, err = os.Open(fh.file.path)
	if err != nil {
		return proto9.Qid{}, err
	}
	return qid, nil
}

func (fh *FileHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	if fh.osfile == nil {
		return 0, server9.ErrFileNotOpen
	}
	n, err := fh.osfile.ReadAt(buf, int64(msg.Offset))
	if n != 0 {
		return uint32(n), nil
	}
	if err == io.EOF {
		return 0, nil
	}
	return 0, err
}

func (fh *FileHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	if fh.osfile == nil {
		return 0, server9.ErrFileNotOpen
	}
	n, err := fh.osfile.WriteAt(msg.Data, int64(msg.Offset))
	return uint32(n), err
}

func (fh *FileHandle) Clunk() error {
	if fh.osfile != nil {
		fh.osfile.Close()
		fh.osfile = nil
	}
	return nil
}

type DirHandle struct {
	file  *File
	stats server9.StatList
}

func (dh *DirHandle) GetFile() (server9.File, error) {
	return dh.file, nil
}

func (dh *DirHandle) GetIounit(maxMessageSize uint32) uint32 {
	return maxMessageSize - proto9.WriteOverhead
}

func (dh *DirHandle) Tcreate(msg *proto9.Tcreate) (server9.Handle, error) {
	newpath := filepath.Join(dh.file.path, msg.Name)
	f := &File{
		parent: dh.file,
		path:   newpath,
	}
	if msg.Perm&proto9.DMDIR != 0 {
		err := os.Mkdir(filepath.Join(dh.file.path, msg.Name), 0644)
		if err != nil {
			return nil, err
		}
		return &DirHandle{
			file: f,
		}, nil
	} else {
		osfile, err := os.Create(filepath.Join(dh.file.path, msg.Name))
		if err != nil {
			return nil, err
		}
		return &FileHandle{
			file:   f,
			osfile: osfile,
		}, nil
	}
}

func (dh *DirHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	return server9.Walk(dh.file, msg.Names)
}

func (dh *DirHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return dh.file.Stat()
}

func (dh *DirHandle) Twstat(msg *proto9.Twstat) error {
	return dh.file.Wstat(msg.Stat)
}

func (dh *DirHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	return dh.file.Qid()
}

func (dh *DirHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	if msg.Offset == 0 {
		dirents, err := ioutil.ReadDir(dh.file.path)
		if err != nil {
			return 0, err
		}
		n := len(dirents)
		stats := make([]proto9.Stat, n, n)
		for i := range dirents {
			stats[i] = osToStat(dirents[i])
		}
		dh.stats = server9.StatList{
			Stats: stats,
		}
	}
	return dh.stats.Tread(msg, buf)
}

func (dh *DirHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	return 0, server9.ErrBadWrite
}

func (dh *DirHandle) Tremove(msg *proto9.Tremove) error {
	dh.file.Remove()
	return dh.Clunk()
}

func (dh *DirHandle) Clunk() error {
	dh.stats = server9.StatList{}
	return nil
}
