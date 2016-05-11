package export

import (
	"acha.ninja/bpy/proto9"
	"acha.ninja/bpy/server9"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

var ErrAuthNotSupported = errors.New("auth not supported")

type ExportServer struct {
	root           string
	rwc            io.ReadWriteCloser
	maxMessageSize uint32
	negMessageSize uint32
	inbuf          []byte
	outbuf         []byte
	qidPathCount   uint64
	fids           map[proto9.Fid]server9.Handle
}

type File struct {
	path   string
	parent *File
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
		return nil, errors.New("unimplemented DirHandle")
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

func (fh *FileHandle) Tcreate(msg *proto9.Tcreate) (server9.File, error) {
	return nil, server9.ErrNotDir
}

func (fh *FileHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	return nil, nil, server9.ErrNotDir
}

func (fh *FileHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return fh.file.Stat()
}

func (fh *FileHandle) Twstat(msg *proto9.Twstat) error {
	return errors.New("unimplemented")
}

func (fh *FileHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	return proto9.Qid{}, errors.New("unimplemented")
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
	return 0, errors.New("unimplemented")
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

func (dh *DirHandle) Tcreate(msg *proto9.Tcreate) (server9.File, error) {
	return nil, errors.New("unimplemented")
}

func (dh *DirHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	return server9.Walk(dh.file, msg.Names)
}

func (dh *DirHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return dh.file.Stat()
}

func (dh *DirHandle) Twstat(msg *proto9.Twstat) error {
	return errors.New("unimplemented")
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

func (dh *DirHandle) Clunk() error {
	dh.stats = server9.StatList{}
	return nil
}

func makeQid(isdir bool) proto9.Qid {
	ty := proto9.QTFILE
	if isdir {
		ty = proto9.QTDIR
	}
	return proto9.Qid{
		Type:    ty,
		Path:    server9.NextPath(),
		Version: uint32(time.Now().UnixNano() / 1000000),
	}
}

func osToStat(ent os.FileInfo) proto9.Stat {
	mode := proto9.FileMode(0777)
	if ent.Mode().IsDir() {
		mode |= proto9.DMDIR
	}
	return proto9.Stat{
		Mode:   mode,
		Atime:  0,
		Mtime:  0,
		Name:   ent.Name(),
		Qid:    makeQid(ent.Mode().IsDir()),
		Length: uint64(ent.Size()),
		UID:    "nobody",
		GID:    "nobody",
		MUID:   "nobody",
	}
}

func (srv *ExportServer) makeroot(path string) (server9.File, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsDir() {
		return nil, server9.ErrNotDir
	}
	return &File{
		path: path,
	}, nil
}

func (srv *ExportServer) AddFid(fid proto9.Fid, fh server9.Handle) error {
	if fid == proto9.NOFID {
		return server9.ErrBadFid
	}
	_, ok := srv.fids[fid]
	if ok {
		return server9.ErrFidInUse
	}
	srv.fids[fid] = fh
	return nil
}

func (srv *ExportServer) handleVersion(msg *proto9.Tversion) proto9.Msg {
	if msg.Tag != proto9.NOTAG {
		return server9.MakeError(msg.Tag, server9.ErrBadTag)
	}
	if msg.MessageSize > srv.maxMessageSize {
		srv.negMessageSize = srv.maxMessageSize
	} else {
		srv.negMessageSize = msg.MessageSize
	}
	srv.inbuf = make([]byte, srv.negMessageSize, srv.negMessageSize)
	srv.outbuf = make([]byte, srv.negMessageSize, srv.negMessageSize)
	return &proto9.Rversion{
		Tag:         msg.Tag,
		MessageSize: srv.negMessageSize,
		Version:     "9P2000",
	}
}

func (srv *ExportServer) handleAttach(msg *proto9.Tattach) proto9.Msg {
	if msg.Afid != proto9.NOFID {
		return server9.MakeError(msg.Tag, ErrAuthNotSupported)
	}

	rootFile, err := srv.makeroot(srv.root)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	fh, err := rootFile.NewHandle()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	err = srv.AddFid(msg.Fid, fh)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	qid, err := rootFile.Qid()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Rattach{
		Tag: msg.Tag,
		Qid: qid,
	}
}

func (srv *ExportServer) handleWalk(msg *proto9.Twalk) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	f, wqids, err := fh.Twalk(msg)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	if f != nil {
		newfh, err := f.NewHandle()
		if msg.NewFid == msg.Fid {
			fh.Clunk()
			delete(srv.fids, msg.Fid)
		}
		err = srv.AddFid(msg.NewFid, newfh)
		if err != nil {
			return server9.MakeError(msg.Tag, err)
		}
		return &proto9.Rwalk{
			Tag:  msg.Tag,
			Qids: wqids,
		}
	}
	return &proto9.Rwalk{
		Tag:  msg.Tag,
		Qids: wqids,
	}
}

func (srv *ExportServer) handleOpen(msg *proto9.Topen) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	qid, err := fh.Topen(msg)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Ropen{
		Tag:    msg.Tag,
		Qid:    qid,
		Iounit: fh.GetIounit(srv.negMessageSize),
	}
}

func (srv *ExportServer) handleCreate(msg *proto9.Tcreate) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	f, err := fh.Tcreate(msg)
	if err != nil {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	qid, err := f.Qid()
	if err != nil {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	newhandle, err := f.NewHandle()
	if err != nil {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	fh.Clunk()
	srv.fids[msg.Fid] = newhandle
	return &proto9.Rcreate{
		Tag:    msg.Tag,
		Qid:    qid,
		Iounit: newhandle.GetIounit(srv.negMessageSize),
	}
}

func (srv *ExportServer) handleRead(msg *proto9.Tread) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	nbytes := uint64(msg.Count)
	maxbytes := uint64(srv.negMessageSize - proto9.ReadOverhead)
	if nbytes > maxbytes {
		nbytes = maxbytes
	}
	buf := make([]byte, nbytes, nbytes)
	n, err := fh.Tread(msg, buf)
	if err != nil {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	return &proto9.Rread{
		Tag:  msg.Tag,
		Data: buf[0:n],
	}
}

func (srv *ExportServer) handleWrite(msg *proto9.Twrite) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	n, err := fh.Twrite(msg)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Rwrite{
		Tag:   msg.Tag,
		Count: uint32(n),
	}
}

/*
func (srv *ExportServer) handleRemove(msg *proto9.Tremove) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	delete(srv.fids, msg.Fid)
	err := fh.Close()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	err = os.Remove(fh.file.path)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Rremove{
		Tag: msg.Tag,
	}
}
*/
func (srv *ExportServer) handleClunk(msg *proto9.Tclunk) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	delete(srv.fids, msg.Fid)
	err := fh.Clunk()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Rclunk{
		Tag: msg.Tag,
	}
}

func (srv *ExportServer) handleStat(msg *proto9.Tstat) proto9.Msg {
	f, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	stat, err := f.Tstat(msg)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Rstat{
		Tag:  msg.Tag,
		Stat: stat,
	}
}

/*
func (srv *ExportServer) handleWStat(msg *proto9.Twstat) proto9.Msg {
	f, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	if strings.Index(msg.Stat.Name, "/") != -1 || strings.Index(msg.Stat.Name, "\\") != -1 {
		return server9.MakeError(msg.Tag, server9.ErrBadPath)
	}
	if msg.Stat.Name == ".." || msg.Stat.Name == "." || msg.Stat.Name == "" {
		return server9.MakeError(msg.Tag, server9.ErrBadPath)
	}
	oldpath := f.file.path
	dir := filepath.Dir(oldpath)
	newpath := filepath.Join(dir, msg.Stat.Name)
	if newpath != oldpath {
		err := os.Rename(oldpath, newpath)
		if err != nil {
			return server9.MakeError(msg.Tag, err)
		}
		f.file.path = newpath
	}
	return &proto9.Rwstat{
		Tag: msg.Tag,
	}
}
*/

func (srv *ExportServer) Serve() error {
	srv.fids = make(map[proto9.Fid]server9.Handle)
	srv.inbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	srv.outbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	for {
		var resp proto9.Msg
		msg, err := proto9.ReadMsg(srv.rwc, srv.inbuf)
		if err != nil {
			return err
		}
		//if verbose {
		//log.Printf("%#v", msg)
		//}
		switch msg := msg.(type) {
		case *proto9.Tversion:
			resp = srv.handleVersion(msg)
		case *proto9.Tattach:
			resp = srv.handleAttach(msg)
		case *proto9.Twalk:
			resp = srv.handleWalk(msg)
		case *proto9.Topen:
			resp = srv.handleOpen(msg)
		//case *proto9.Tread:
		//	resp = srv.handleRead(msg)
		case *proto9.Tclunk:
			resp = srv.handleClunk(msg)
		case *proto9.Tstat:
			resp = srv.handleStat(msg)
		//case *proto9.Twstat:
		//	resp = srv.handleWStat(msg)
		//case *proto9.Twrite:
		//	resp = srv.handleWrite(msg)
		//case *proto9.Tremove:
		//	resp = srv.handleRemove(msg)
		case *proto9.Tauth:
			resp = server9.MakeError(msg.Tag, ErrAuthNotSupported)
		case *proto9.Tcreate:
			resp = srv.handleCreate(msg)
		default:
			return errors.New("bad message")
		}
		//if verbose {
		//log.Printf("%#v", resp)
		//}
		err = proto9.WriteMsg(srv.rwc, srv.outbuf, resp)
		if err != nil {
			return err
		}
	}
}

func NewExportServer(rwc io.ReadWriteCloser, path string) (*ExportServer, error) {
	root, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	srv := &ExportServer{
		root:           root,
		rwc:            rwc,
		maxMessageSize: 131072,
	}
	return srv, nil
}
