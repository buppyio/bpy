package remote

import (
	"acha.ninja/bpy/proto9"
	"acha.ninja/bpy/server9"
	"errors"
	"io"
	//"log"
	"os"
	"path/filepath"
	"time"
)

const TAGDBNAME = "tags.db"

var ErrAuthNotSupported = errors.New("auth not supported")

type Server struct {
	root           server9.File
	rwc            io.ReadWriteCloser
	maxMessageSize uint32
	negMessageSize uint32
	inbuf          []byte
	outbuf         []byte
	qidPathCount   uint64
	fids           map[proto9.Fid]server9.Handle
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

func (srv *Server) AddFid(fid proto9.Fid, fh server9.Handle) error {
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

func (srv *Server) handleVersion(msg *proto9.Tversion) proto9.Msg {
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

func (srv *Server) handleAttach(msg *proto9.Tattach) proto9.Msg {
	if msg.Afid != proto9.NOFID {
		return server9.MakeError(msg.Tag, ErrAuthNotSupported)
	}

	fh, err := srv.root.NewHandle()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	err = srv.AddFid(msg.Fid, fh)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	qid, err := srv.root.Qid()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Rattach{
		Tag: msg.Tag,
		Qid: qid,
	}
}

func (srv *Server) handleWalk(msg *proto9.Twalk) proto9.Msg {
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

func (srv *Server) handleOpen(msg *proto9.Topen) proto9.Msg {
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

func (srv *Server) handleCreate(msg *proto9.Tcreate) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	if !server9.ValidFileName(msg.Name) {
		return server9.MakeError(msg.Tag, server9.ErrBadPath)
	}
	newhandle, err := fh.Tcreate(msg)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	f, err := newhandle.GetFile()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	qid, err := f.Qid()
	if err != nil {
		return server9.MakeError(msg.Tag, err)
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

func (srv *Server) handleWrite(msg *proto9.Twrite) proto9.Msg {
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

func (srv *Server) handleRemove(msg *proto9.Tremove) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	delete(srv.fids, msg.Fid)
	err := fh.Tremove(msg)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Rremove{
		Tag: msg.Tag,
	}
}

func (srv *Server) handleClunk(msg *proto9.Tclunk) proto9.Msg {
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

func (srv *Server) handleStat(msg *proto9.Tstat) proto9.Msg {
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

func (srv *Server) handleWStat(msg *proto9.Twstat) proto9.Msg {
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return server9.MakeError(msg.Tag, server9.ErrNoSuchFid)
	}
	if !server9.ValidFileName(msg.Stat.Name) {
		return server9.MakeError(msg.Tag, server9.ErrBadPath)
	}
	err := fh.Twstat(msg)
	if err != nil {
		return server9.MakeError(msg.Tag, err)
	}
	return &proto9.Rwstat{
		Tag: msg.Tag,
	}
}

func (srv *Server) Serve() error {
	srv.fids = make(map[proto9.Fid]server9.Handle)
	srv.inbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	srv.outbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	for {
		var resp proto9.Msg
		msg, err := proto9.ReadMsg(srv.rwc, srv.inbuf)
		if err != nil {
			return err
		}
		//log.Printf("%#v", msg)
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
			resp = server9.MakeError(msg.Tag, ErrAuthNotSupported)
		case *proto9.Tcreate:
			resp = srv.handleCreate(msg)
		default:
			return errors.New("bad message")
		}
		//log.Printf("%#v", resp)
		err = proto9.WriteMsg(srv.rwc, srv.outbuf, resp)
		if err != nil {
			return err
		}
	}
}

func NewServer(rwc io.ReadWriteCloser, packDir string) (*Server, error) {
	packDir, err := filepath.Abs(packDir)
	if err != nil {
		return nil, err
	}

	root := &RootFile{
		packDir: &File{
			parent: nil,
			path:   packDir,
		},
		ctlFile: &CtlFile{
			dbPath: filepath.Join(packDir, TAGDBNAME),
		},
	}
	srv := &Server{
		root:           root,
		rwc:            rwc,
		maxMessageSize: 131072,
	}
	return srv, nil
}
