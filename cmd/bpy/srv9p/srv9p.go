package srv9p

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/proto9"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"
	"os"
)

var (
	ErrNoSuchFid        = errors.New("no such fid")
	ErrFidInUse         = errors.New("fid in use")
	ErrBadFid           = errors.New("bad fid")
	ErrAuthNotSupported = errors.New("auth not supported")
	ErrBadTag           = errors.New("bad tag")
	ErrBadPath          = errors.New("bad path")
	ErrNotDir           = errors.New("not a directory path")
	ErrNotExist         = errors.New("no such file")
	ErrFileNotOpen      = errors.New("file not open")
	ErrBadReadOffset    = errors.New("bad read offset")
	ErrReadOnly         = errors.New("read only")
)

type File struct {
	parent  *File
	dirhash [32]byte
	diridx  int
	dirent  fs.DirEnt
	stat    proto9.Stat
}

type FileHandle struct {
	file *File
	rdr  *fs.FileReader

	diroffset uint64
	stats     []proto9.Stat
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

func (srv *proto9Server) dirEntToStat(ent *fs.DirEnt) proto9.Stat {
	mode := proto9.FileMode(0777)
	qtype := proto9.QidType(0)
	if ent.Mode.IsDir() {
		mode |= proto9.DMDIR
		qtype |= proto9.QTDIR
	} else {
		qtype |= proto9.QTFILE
	}
	return proto9.Stat{
		Mode:   mode,
		Atime:  0,
		Mtime:  0,
		Name:   ent.Name,
		Qid:    srv.makeQid(*ent),
		Length: uint64(ent.Size),
		UID:    "nobody",
		GID:    "nobody",
		MUID:   "nobody",
	}
}

func (srv *proto9Server) fsStatToProto9Stat(dir fs.DirEnts) []proto9.Stat {
	n := len(dir)
	stats := make([]proto9.Stat, n, n)
	for i := range dir {
		stats[i] = srv.dirEntToStat(&dir[i])
	}
	return stats
}

func (srv *proto9Server) makeQid(ent fs.DirEnt) proto9.Qid {
	path := srv.qidPathCount
	srv.qidPathCount++
	ty := proto9.QTFILE
	if ent.Mode.IsDir() {
		ty = proto9.QTDIR
	}
	return proto9.Qid{
		Type:    ty,
		Path:    path,
		Version: 0,
	}
}

func (srv *proto9Server) makeRoot(hash [32]byte) (*File, error) {
	ents, err := fs.ReadDir(srv.store, hash)
	if err != nil {
		return nil, err
	}
	return &File{
		dirhash: hash,
		dirent:  ents[0],
		stat:    srv.dirEntToStat(&ents[0]),
	}, nil
}

func (srv *proto9Server) AddFid(fid proto9.Fid, fh *FileHandle) error {
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

func (srv *proto9Server) ClunkFid(fid proto9.Fid) error {
	f, ok := srv.fids[fid]
	if !ok {
		return ErrNoSuchFid
	}
	err := f.Close()
	if err != nil {
		return err
	}
	delete(srv.fids, fid)
	return nil
}

func (srv *proto9Server) readMsg(c net.Conn) (proto9.Msg, error) {
	if len(srv.inbuf) < 5 {
		return nil, proto9.ErrBuffTooSmall
	}
	_, err := c.Read(srv.inbuf[0:5])
	if err != nil {
		return nil, err
	}
	sz := int(binary.LittleEndian.Uint16(srv.inbuf[0:4]))
	if len(srv.inbuf) < sz {
		return nil, proto9.ErrBuffTooSmall
	}
	_, err = c.Read(srv.inbuf[5:sz])
	if err != nil {
		return nil, err
	}
	return proto9.UnpackMsg(srv.inbuf[0:sz])
}

func (srv *proto9Server) sendMsg(c net.Conn, msg proto9.Msg) error {
	packed, err := proto9.PackMsg(srv.outbuf, msg)
	if err != nil {
		return err
	}
	_, err = c.Write(packed)
	if err != nil {
		return err
	}
	return nil
}

func (srv *proto9Server) handleVersion(msg *proto9.Tversion) proto9.Msg {
	if msg.Tag != proto9.NOTAG {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: ErrBadTag.Error(),
		}
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
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: ErrAuthNotSupported.Error(),
		}
	}

	rootFile, err := srv.makeRoot(srv.Root)
	if err != nil {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: err.Error(),
		}
	}
	f := &FileHandle{
		file: rootFile,
	}
	err = srv.AddFid(msg.Fid, f)
	if err != nil {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: err.Error(),
		}
	}

	return &proto9.Rattach{
		Tag: msg.Tag,
		Qid: rootFile.stat.Qid,
	}
}

func (srv *proto9Server) handleWalk(msg *proto9.Twalk) proto9.Msg {
	var werr error

	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: ErrNoSuchFid.Error(),
		}
	}
	f := fh.file

	wqids := make([]proto9.Qid, 0, len(msg.Names))

	i := 0
	name := ""
	for i, name = range msg.Names {
		found := false
		if name == "." || name == "" || strings.Index(name. "/") != -1 {
			return &proto9.Rerror{
				Tag: msg.Tag,
				Err: ErrBadPath.Error(),
			}
		}
		if name == ".." {
			f = f.parent
			if f == nil {
				werr = ErrBadPath
				goto walkerr
			}
			wqids = append(wqids, f.stat.Qid)
			continue
		}

		if !f.dirent.Mode.IsDir() {
			goto walkerr
		}

		dirents, err := fs.ReadDir(srv.store, f.dirent.Data)
		if err != nil {
			werr = err
			goto walkerr
		}
		for diridx := range dirents {
			if dirents[diridx].Name == name {
				found = true
				f = &File{
					parent:  f,
					dirhash: f.dirent.Data,
					diridx:  diridx,
					dirent:  dirents[diridx],
					stat:    srv.dirEntToStat(&dirents[diridx]),
				}
				wqids = append(wqids, f.stat.Qid)
				break
			}
		}
		if !found {
			werr = ErrNotExist
			goto walkerr
		}
	}

	if msg.NewFid == msg.Fid {
		fh.Close()
		delete(srv.fids, msg.Fid)
	}
	fh = &FileHandle{
		file: f,
	}
	werr = srv.AddFid(msg.NewFid, fh)
	if werr != nil {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: werr.Error(),
		}
	}
	return &proto9.Rwalk{
		Tag:  msg.Tag,
		Qids: wqids,
	}

walkerr:
	if i == 0 {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: werr.Error(),
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
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: ErrNoSuchFid.Error(),
		}
	}
	if fh.file.dirent.Mode.IsRegular() {
		rdr, err := fs.Open(srv.store, fh.file.dirhash, fh.file.dirent.Name)
		if err != nil {
			return &proto9.Rerror{
				Tag: msg.Tag,
				Err: err.Error(),
			}
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
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: ErrNoSuchFid.Error(),
		}
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
				return &proto9.Rerror{
					Tag: msg.Tag,
					Err: err.Error(),
				}
			}
			fh.stats = srv.fsStatToProto9Stat(dirents)
			fh.diroffset = 0
		}

		if msg.Offset != fh.diroffset {
			return &proto9.Rerror{
				Tag: msg.Tag,
				Err: ErrBadReadOffset.Error(),
			}
		}

		n := 0
		for {
			if len(fh.stats) == 0 {
				break
			}
			curstat := fh.stats[0]
			statlen := proto9.StatLen(&curstat)
			if uint64(statlen+n) > nbytes {
				if n == 0 {
					return &proto9.Rerror{
						Tag: msg.Tag,
						Err: proto9.ErrBuffTooSmall.Error(),
					}
				}
				break
			}
			proto9.PackStat(buf[n:n+statlen], &curstat)
			n += statlen
			fh.stats = fh.stats[1:]
		}
		fh.diroffset += uint64(n)
		return &proto9.Rread{
			Tag:  msg.Tag,
			Data: buf[0:n],
		}

	} else {

		if fh.rdr == nil {
			return &proto9.Rerror{
				Tag: msg.Tag,
				Err: ErrFileNotOpen.Error(),
			}
		}

		n, err := fh.rdr.ReadAt(buf, int64(msg.Offset))
		if err != nil && err != io.EOF {
			return &proto9.Rerror{
				Tag: msg.Tag,
				Err: err.Error(),
			}
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
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: ErrNoSuchFid.Error(),
		}
	}
	delete(srv.fids, msg.Fid)
	err := f.Close()
	if err != nil {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: "unimplemented read",
		}
	}
	return &proto9.Rclunk{
		Tag: msg.Tag,
	}
}

func (srv *proto9Server) handleStat(msg *proto9.Tstat) proto9.Msg {
	f, ok := srv.fids[msg.Fid]
	if !ok {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: ErrNoSuchFid.Error(),
		}
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
		msg, err := srv.readMsg(c)
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
			resp = &proto9.Rerror{
				Tag: msg.Tag,
				Err: ErrAuthNotSupported.Error(),
			}
		case *proto9.Twrite:
			resp = &proto9.Rerror{
				Tag: msg.Tag,
				Err: ErrReadOnly.Error(),
			}
		case *proto9.Twstat:
			resp = &proto9.Rerror{
				Tag: msg.Tag,
				Err: ErrReadOnly.Error(),
			}
		default:
			log.Println("unhandled message type")
			return
		}
		log.Printf("%#v", resp)
		err = srv.sendMsg(c, resp)
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
