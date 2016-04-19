package remote

import (
	"acha.ninja/bpy/proto9"
	"acha.ninja/bpy/server9"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
)

var (
	ErrAuthNotSupported = errors.New("auth not supported")
)

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
	return makeQid(stat), nil
}

type FileHandle struct {
	file *File

	isopen bool
	isdir  bool
	osfile *os.File
	stats  server9.StatList
}

func (f *FileHandle) Close() error {
	f.isopen = false
	if f.osfile != nil {
		err := f.osfile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

type proto9Server struct {
	Root           string
	maxMessageSize uint32
	negMessageSize uint32
	inbuf          []byte
	outbuf         []byte
	qidPathCount   uint64
	fids           map[proto9.Fid]*FileHandle
}

func makeQid(ent os.FileInfo) proto9.Qid {
	ty := proto9.QTFILE
	if ent.Mode().IsDir() {
		ty = proto9.QTDIR
	}
	return proto9.Qid{
		Type:    ty,
		Path:    server9.NextPath(),
		Version: uint32(ent.ModTime().UnixNano() / 1000000),
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
		Qid:    makeQid(ent),
		Length: uint64(ent.Size()),
		UID:    "nobody",
		GID:    "nobody",
		MUID:   "nobody",
	}
}

func (srv *proto9Server) makeRoot(path string) (*File, error) {
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

func (srv *proto9Server) AddFid(fid proto9.Fid, fh *FileHandle) error {
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

func (srv *proto9Server) ClunkFid(fid proto9.Fid) error {
	f, ok := srv.fids[fid]
	if !ok {
		return server9.ErrNoSuchFid
	}
	err := f.Close()
	if err != nil {
		return err
	}
	delete(srv.fids, fid)
	return nil
}

func (srv *proto9Server) handleVersion(msg *proto9.Tversion) proto9.Msg {
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

func (srv *proto9Server) handleAttach(msg *proto9.Tattach) proto9.Msg {
	if msg.Afid != proto9.NOFID {
		return server9.MakeError(msg.Tag, ErrAuthNotSupported)
	}

	rootFile, err := srv.makeRoot(srv.Root)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	f := &FileHandle{
		file: rootFile,
	}
	err = srv.AddFid(msg.Fid, f)
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

func (srv *proto9Server) handleWalk(msg *proto9.Twalk) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	f, wqids, err := server9.Walk(fh.file, msg.Names)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	if f != nil {
		if msg.NewFid == msg.Fid {
			fh.Close()
			delete(srv.fids, msg.Fid)
		}
		fh = &FileHandle{
			file: f.(*File),
		}
		err = srv.AddFid(msg.NewFid, fh)
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

func (srv *proto9Server) handleOpen(msg *proto9.Topen) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	stat, err := fh.file.Stat()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	if stat.Qid.IsFile() {
		fh.isdir = false
		rdr, err := os.Open(fh.file.path)
		if err != nil {
			return server9.MakeError(msg.Tag, err)
		}
		fh.osfile = rdr
	} else {
		fh.isdir = true
	}
	fh.isopen = true
	return &proto9.Ropen{
		Tag:    msg.Tag,
		Qid:    stat.Qid,
		Iounit: srv.negMessageSize - proto9.WriteOverhead,
	}
}

func (srv *proto9Server) handleRead(msg *proto9.Tread) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	if !fh.isopen {
		return server9.MakeError(msg.Tag, server9.ErrFileNotOpen)
	}
	nbytes := uint64(msg.Count)
	maxbytes := uint64(srv.negMessageSize - proto9.ReadOverhead)
	if nbytes > maxbytes {
		nbytes = maxbytes
	}
	buf := make([]byte, nbytes, nbytes)

	if fh.isdir {
		if msg.Offset == 0 {
			dirents, err := ioutil.ReadDir(fh.file.path)
			if err != nil {
				return server9.MakeError(msg.Tag, err)
			}
			n := len(dirents)
			stats := make([]proto9.Stat, n, n)
			for i := range dirents {
				stats[i] = osToStat(dirents[i])
			}
			fh.stats = server9.StatList{
				Stats: stats,
			}
		}
		n, err := fh.stats.ReadAt(buf, msg.Offset)
		if err != nil {
			return server9.MakeError(msg.Tag, err)
		}
		return &proto9.Rread{
			Tag:  msg.Tag,
			Data: buf[0:n],
		}

	} else {
		if fh.osfile == nil {
			return server9.MakeError(msg.Tag, errors.New("internal error"))
		}
		n, err := fh.osfile.ReadAt(buf, int64(msg.Offset))
		if err != nil && err != io.EOF {
			return server9.MakeError(msg.Tag, err)
		}
		return &proto9.Rread{
			Tag:  msg.Tag,
			Data: buf[0:n],
		}
	}
}

func (srv *proto9Server) handleClunk(msg *proto9.Tclunk) proto9.Msg {
	f, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	delete(srv.fids, msg.Fid)
	err := f.Close()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Rclunk{
		Tag: msg.Tag,
	}
}

func (srv *proto9Server) handleStat(msg *proto9.Tstat) proto9.Msg {
	f, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	stat, err := f.file.Stat()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Rstat{
		Tag:  msg.Tag,
		Stat: stat,
	}
}

func (srv *proto9Server) serveConn(c net.Conn) {
	defer c.Close()
	srv.fids = make(map[proto9.Fid]*FileHandle)
	srv.inbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	srv.outbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	for {
		var resp proto9.Msg
		msg, err := server9.ReadMsg(c, srv.inbuf)
		if err != nil {
			log.Printf("error reading message: %s", err.Error())
			return
		}
		log.Printf("%#v", msg)
		switch msg := msg.(type) {
		case *proto9.Tversion:
			resp = srv.handleVersion(msg)
		case *proto9.Tattach:
			resp = srv.handleAttach(msg)
		case *proto9.Twalk:
			resp = srv.handleWalk(msg)
		case *proto9.Topen:
			resp = srv.handleOpen(msg)
		case *proto9.Tread:
			resp = srv.handleRead(msg)
		case *proto9.Tclunk:
			resp = srv.handleClunk(msg)
		case *proto9.Tstat:
			resp = srv.handleStat(msg)
		case *proto9.Tauth:
			resp = server9.MakeError(msg.Tag, ErrAuthNotSupported)
		case *proto9.Twrite:
			resp = server9.MakeError(msg.Tag, errors.New("unimplemented"))
		case *proto9.Twstat:
			resp = server9.MakeError(msg.Tag, errors.New("unimplemented"))
		case *proto9.Tcreate:
			resp = server9.MakeError(msg.Tag, errors.New("unimplemented"))
		default:
			log.Println("unhandled message type")
			return
		}
		log.Printf("%#v", resp)
		err = server9.WriteMsg(c, srv.outbuf, resp)
		if err != nil {
			log.Printf("error sending message: %s", err.Error())
			return
		}
	}
}

func Remote() {
	root := "/home/ac/.bpy/store"
	log.Println("Serving 9p...")
	l, err := net.Listen("tcp", "127.0.0.1:9001")
	if err != nil {
		log.Fatal(err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		srv := &proto9Server{
			Root:           root,
			maxMessageSize: 4096,
		}
		go srv.serveConn(c)
	}
}