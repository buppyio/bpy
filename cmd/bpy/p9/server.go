package p9

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/p9/proto9"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/remote/client"
	"errors"
	"io"
	"log"
	"strings"
	"time"
)

var (
	ErrNoSuchFid        = errors.New("no such fid")
	ErrFidInUse         = errors.New("fid in use")
	ErrBadFid           = errors.New("bad fid")
	ErrBadTag           = errors.New("bad tag")
	ErrBadPath          = errors.New("bad path")
	ErrNotDir           = errors.New("not a directory path")
	ErrNotExist         = errors.New("no such file")
	ErrFileNotOpen      = errors.New("file not open")
	ErrFileAlreadyOpen  = errors.New("file already open")
	ErrBadRead          = errors.New("bad read")
	ErrBadWrite         = errors.New("bad write")
	ErrAuthNotSupported = errors.New("auth not supported")
	ErrInvalidMount     = errors.New("invalid mount")
)

type File interface {
	Parent() (File, error)
	Child(name string) (File, error)
	Qid() (proto9.Qid, error)
	Stat() (proto9.Stat, error)
	NewHandle() (Handle, error)
}

type Handle interface {
	GetFile() (File, error)
	GetIounit(maxMessageSize uint32) uint32
	Twalk(msg *proto9.Twalk) (File, []proto9.Qid, error)
	Topen(msg *proto9.Topen) (proto9.Qid, error)
	Tread(msg *proto9.Tread, buf []byte) (uint32, error)
	Twrite(msg *proto9.Twrite) (uint32, error)
	Tcreate(msg *proto9.Tcreate) (Handle, error)
	Twstat(msg *proto9.Twstat) error
	Tremove(msg *proto9.Tremove) error
	Tstat(msg *proto9.Tstat) (proto9.Stat, error)
	Clunk() error
}

type Server struct {
	rwc            io.ReadWriteCloser
	maxMessageSize uint32
	negMessageSize uint32
	inbuf          []byte
	outbuf         []byte
	fids           map[proto9.Fid]Handle
	client         *client.Client
	store          bpy.CStore
	root           [32]byte
}

func makeQid(isdir bool) proto9.Qid {
	ty := proto9.QTFILE
	if isdir {
		ty = proto9.QTDIR
	}
	return proto9.Qid{
		Type:    ty,
		Path:    nextPath(),
		Version: uint32(time.Now().UnixNano() / 1000000),
	}
}

func (srv *Server) AddFid(fid proto9.Fid, fh Handle) error {
	if fid == proto9.NOFID {
		return ErrBadFid
	}
	_, ok := srv.fids[fid]
	if ok {
		return ErrFidInUse
	}
	srv.fids[fid] = fh
	return nil
}

func (srv *Server) handleVersion(msg *proto9.Tversion) proto9.Msg {
	if msg.Tag != proto9.NOTAG {
		return proto9.MakeError(msg.Tag, ErrBadTag)
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

func (srv *Server) handleAttach(msg *proto9.Tattach) proto9.Msg {
	if msg.Afid != proto9.NOFID {
		return proto9.MakeError(msg.Tag, ErrAuthNotSupported)
	}

	rootDirEnt, err := fs.Walk(srv.store, srv.root, "/")
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}

	root := &file{
		srv:    srv,
		parent: nil,
		qid:    makeQid(true),
		path:   "/",
		dirEnt: rootDirEnt,
	}

	fh, err := root.NewHandle()
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	err = srv.AddFid(msg.Fid, fh)
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	qid, err := root.Qid()
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	return &proto9.Rattach{
		Tag: msg.Tag,
		Qid: qid,
	}
}

func (srv *Server) handleWalk(msg *proto9.Twalk) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return proto9.MakeError(msg.Tag, ErrNoSuchFid)
	}
	f, wqids, err := fh.Twalk(msg)
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	if f != nil {
		newfh, err := f.NewHandle()
		if err != nil {
			return proto9.MakeError(msg.Tag, err)
		}
		if msg.NewFid == msg.Fid {
			fh.Clunk()
			delete(srv.fids, msg.Fid)
		}
		err = srv.AddFid(msg.NewFid, newfh)
		if err != nil {
			return proto9.MakeError(msg.Tag, err)
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

func (srv *Server) handleOpen(msg *proto9.Topen) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return proto9.MakeError(msg.Tag, ErrNoSuchFid)
	}
	qid, err := fh.Topen(msg)
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	return &proto9.Ropen{
		Tag:    msg.Tag,
		Qid:    qid,
		Iounit: fh.GetIounit(srv.negMessageSize),
	}
}

func validFileName(name string) bool {
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	if name == ".." || name == "." || name == "" {
		return false
	}
	return true
}

func (srv *Server) handleCreate(msg *proto9.Tcreate) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return proto9.MakeError(msg.Tag, ErrNoSuchFid)
	}
	if !validFileName(msg.Name) {
		return proto9.MakeError(msg.Tag, ErrBadPath)
	}
	newhandle, err := fh.Tcreate(msg)
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	f, err := newhandle.GetFile()
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	qid, err := f.Qid()
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	fh.Clunk()
	srv.fids[msg.Fid] = newhandle
	return &proto9.Rcreate{
		Tag:    msg.Tag,
		Qid:    qid,
		Iounit: newhandle.GetIounit(srv.negMessageSize),
	}
}

func (srv *Server) handleRead(msg *proto9.Tread) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return proto9.MakeError(msg.Tag, ErrNoSuchFid)
	}
	nbytes := uint64(msg.Count)
	maxbytes := uint64(srv.negMessageSize - proto9.ReadOverhead)
	if nbytes > maxbytes {
		nbytes = maxbytes
	}
	buf := make([]byte, nbytes, nbytes)
	n, err := fh.Tread(msg, buf)
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	return &proto9.Rread{
		Tag:  msg.Tag,
		Data: buf[0:n],
	}
}

func (srv *Server) handleWrite(msg *proto9.Twrite) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return proto9.MakeError(msg.Tag, ErrNoSuchFid)
	}
	n, err := fh.Twrite(msg)
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	return &proto9.Rwrite{
		Tag:   msg.Tag,
		Count: uint32(n),
	}
}

func (srv *Server) handleRemove(msg *proto9.Tremove) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return proto9.MakeError(msg.Tag, ErrNoSuchFid)
	}
	delete(srv.fids, msg.Fid)
	err := fh.Tremove(msg)
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	return &proto9.Rremove{
		Tag: msg.Tag,
	}
}

func (srv *Server) handleClunk(msg *proto9.Tclunk) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return proto9.MakeError(msg.Tag, ErrNoSuchFid)
	}
	delete(srv.fids, msg.Fid)
	err := fh.Clunk()
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	return &proto9.Rclunk{
		Tag: msg.Tag,
	}
}

func (srv *Server) handleStat(msg *proto9.Tstat) proto9.Msg {
	f, ok := srv.fids[msg.Fid]
	if !ok {
		return proto9.MakeError(msg.Tag, ErrNoSuchFid)
	}
	stat, err := f.Tstat(msg)
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	return &proto9.Rstat{
		Tag:  msg.Tag,
		Stat: stat,
	}
}

func (srv *Server) handleWStat(msg *proto9.Twstat) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return proto9.MakeError(msg.Tag, ErrNoSuchFid)
	}
	if !validFileName(msg.Stat.Name) {
		return proto9.MakeError(msg.Tag, ErrBadPath)
	}
	err := fh.Twstat(msg)
	if err != nil {
		return proto9.MakeError(msg.Tag, err)
	}
	return &proto9.Rwstat{
		Tag: msg.Tag,
	}
}

func (srv *Server) Serve() error {
	srv.fids = make(map[proto9.Fid]Handle)
	srv.inbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	srv.outbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	for {
		var resp proto9.Msg
		msg, err := proto9.ReadMsg(srv.rwc, srv.inbuf)
		if err != nil {
			return err
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
		case *proto9.Twstat:
			resp = srv.handleWStat(msg)
		case *proto9.Twrite:
			resp = srv.handleWrite(msg)
		case *proto9.Tremove:
			resp = srv.handleRemove(msg)
		case *proto9.Tauth:
			resp = proto9.MakeError(msg.Tag, ErrAuthNotSupported)
		case *proto9.Tcreate:
			resp = srv.handleCreate(msg)
		default:
			return errors.New("bad message")
		}
		log.Printf("%#v", resp)
		err = proto9.WriteMsg(srv.rwc, srv.outbuf, resp)
		if err != nil {
			return err
		}
	}
}
