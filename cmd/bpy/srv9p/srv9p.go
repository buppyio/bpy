package srv9p

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/proto9"
	"acha.ninja/bpy/server9"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"
	"os"
)

var (
	ErrReadOnly         = errors.New("read only")
	ErrAuthNotSupported = errors.New("auth not supported")
)

type File struct {
	store   bpy.CStoreReader
	parent  *File
	dirhash [32]byte
	diridx  int
	dirent  fs.DirEnt
	stat    proto9.Stat
}

func (f *File) Parent() (server9.File, error) {
	if f.parent == nil {
		return nil, server9.ErrBadPath
	}
	return f.parent, nil
}

func (f *File) Child(name string) (server9.File, error) {
	if !f.dirent.Mode.IsDir() {
		return nil, server9.ErrNotDir
	}

	dirents, err := fs.ReadDir(f.store, f.dirhash)

	if err != nil {
		return nil, err
	}

	for idx, ent := range dirents {
		if ent.Name == name {
			return &File{
				store:   f.store,
				parent:  f,
				dirhash: f.dirent.Data,
				diridx:  idx,
				dirent:  ent,
				stat:    dirEntToStat(&ent),
			}, nil
		}
	}
	return nil, server9.ErrNotExist
}

func (f *File) Stat() (proto9.Stat, error) {
	return f.stat, nil
}

func (f *File) Qid() (proto9.Qid, error) {
	return f.stat.Qid, nil
}

type FileHandle struct {
	file *File

	rdr *fs.FileReader

	stats server9.StatList
}

func (f *FileHandle) Close() error {
	if f.rdr != nil {
		err := f.rdr.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

type proto9Server struct {
	Root           [32]byte
	store          bpy.CStoreReader
	maxMessageSize uint32
	negMessageSize uint32
	inbuf          []byte
	outbuf         []byte
	qidPathCount   uint64
	fids           map[proto9.Fid]*FileHandle
}

func makeQid(ent *fs.DirEnt) proto9.Qid {
	ty := proto9.QTFILE
	if ent.Mode.IsDir() {
		ty = proto9.QTDIR
	}
	return proto9.Qid{
		Type:    ty,
		Path:    server9.NextPath(),
		Version: 0,
	}
}

func dirEntToStat(ent *fs.DirEnt) proto9.Stat {
	mode := proto9.FileMode(0777)
	if ent.Mode.IsDir() {
		mode |= proto9.DMDIR
	}
	return proto9.Stat{
		Mode:   mode,
		Atime:  0,
		Mtime:  0,
		Name:   ent.Name,
		Qid:    makeQid(ent),
		Length: uint64(ent.Size),
		UID:    "nobody",
		GID:    "nobody",
		MUID:   "nobody",
	}
}

func (srv *proto9Server) makeRoot(hash [32]byte) (*File, error) {
	ents, err := fs.ReadDir(srv.store, hash)
	if err != nil {
		return nil, err
	}
	return &File{
		store:   srv.store,
		dirhash: hash,
		diridx:  0,
		dirent:  ents[0],
		stat:    dirEntToStat(&ents[0]),
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

	return &proto9.Rattach{
		Tag: msg.Tag,
		Qid: rootFile.stat.Qid,
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
	if fh.file.dirent.Mode.IsRegular() {
		rdr, err := fs.Open(srv.store, fh.file.dirhash, fh.file.dirent.Name)
		if err != nil {
			return server9.MakeError(msg.Tag, err)
		}
		fh.rdr = rdr
	}
	return &proto9.Ropen{
		Tag:    msg.Tag,
		Qid:    fh.file.stat.Qid,
		Iounit: srv.negMessageSize - proto9.WriteOverhead,
	}
}

func (srv *proto9Server) handleRead(msg *proto9.Tread) proto9.Msg {
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

	if fh.file.dirent.Mode.IsDir() {
		if msg.Offset == 0 {
			dirents, err := fs.ReadDir(srv.store, fh.file.dirent.Data)
			if err != nil {
				return server9.MakeError(msg.Tag, err)
			}
			n := len(dirents)
			stats := make([]proto9.Stat, n, n)
			for i := range dirents {
				stats[i] = dirEntToStat(&dirents[i])
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
		if fh.rdr == nil {
			return server9.MakeError(msg.Tag, server9.ErrFileNotOpen)
		}
		n, err := fh.rdr.ReadAt(buf, int64(msg.Offset))
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
	return &proto9.Rstat{
		Tag:  msg.Tag,
		Stat: f.file.stat,
	}
}

func (srv *proto9Server) serveConn(c net.Conn) {
	defer c.Close()
	srv.fids = make(map[proto9.Fid]*FileHandle)
	srv.inbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	srv.outbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	for {
		var resp proto9.Msg
		msg, err := proto9.ReadMsg(c, srv.inbuf)
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
			resp = server9.MakeError(msg.Tag, ErrReadOnly)
		case *proto9.Tremove:
			resp = server9.MakeError(msg.Tag, ErrReadOnly)
		case *proto9.Twstat:
			resp = server9.MakeError(msg.Tag, ErrReadOnly)
		case *proto9.Tcreate:
			resp = server9.MakeError(msg.Tag, ErrReadOnly)
		default:
			log.Println("unhandled message type")
			return
		}
		log.Printf("%#v", resp)
		err = proto9.WriteMsg(c, srv.outbuf, resp)
		if err != nil {
			log.Printf("error sending message: %s", err.Error())
			return
		}
	}

}

func Srv9p() {

	var hash [32]byte
	store, err := cstore.NewReader("/home/ac/.bpy/store", "/home/ac/.bpy/cache")
	if err != nil {
		panic(err)
	}
	hbytes, err := hex.DecodeString(os.Args[2])
	if err != nil {
		panic(err)
	}
	copy(hash[:], hbytes)

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
			store:          store,
			Root:           hash,
			maxMessageSize: 4096,
		}
		go srv.serveConn(c)
	}
}
