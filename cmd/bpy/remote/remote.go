package remote

import (
	"acha.ninja/bpy/proto9"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
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
	parent *File
	path   string
	qid    proto9.Qid
}

type FileHandle struct {
	file *File

	hostfile *os.File

	diroffset uint64
	stats     []proto9.Stat
}

func (f *FileHandle) Close() error {
	if f.hostfile != nil {
		err := f.hostfile.Close()
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

func (srv *proto9Server) dirEntToStat(ent os.FileInfo) proto9.Stat {
	mode := proto9.FileMode(0777)
	qtype := proto9.QidType(0)
	if ent.Mode().IsDir() {
		mode |= proto9.DMDIR
		qtype |= proto9.QTDIR
	} else {
		qtype |= proto9.QTFILE
	}
	return proto9.Stat{
		Mode:   mode,
		Atime:  0,
		Mtime:  0,
		Name:   ent.Name(),
		Qid:    srv.makeQid(ent.Mode().IsDir()),
		Length: uint64(ent.Size()),
		UID:    "nobody",
		GID:    "nobody",
		MUID:   "nobody",
	}
}

func (srv *proto9Server) makeQid(isdir bool) proto9.Qid {
	path := srv.qidPathCount
	srv.qidPathCount++
	ty := proto9.QTFILE
	if isdir {
		ty = proto9.QTDIR
	}
	return proto9.Qid{
		Type:    ty,
		Path:    path,
		Version: 0,
	}
}

func (srv *proto9Server) makeRoot(path string) (*File, error) {
	stat, err := os.Stat(srv.Root)
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsDir() {
		return nil, ErrNotDir
	}
	return &File{
		path: path,
		qid:  srv.makeQid(true),
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
		Qid: rootFile.qid,
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
		if name == "." || name == "" || strings.Index(name, "/") != -1 {
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
			wqids = append(wqids, f.qid)
			continue
		}
		stat, err := os.Stat(f.path)
		if err != nil {
			werr = err
			goto walkerr
		}
		if !stat.Mode().IsDir() {
			werr = ErrNotDir
			goto walkerr
		}

		dirents, err := ioutil.ReadDir(f.path)
		if err != nil {
			werr = err
			goto walkerr
		}
		for diridx := range dirents {
			if dirents[diridx].Name() == name {
				found = true
				f = &File{
					parent: f,
					path:   filepath.Join(f.path, name),
					qid:    srv.makeQid(dirents[diridx].IsDir()),
				}
				wqids = append(wqids, f.qid)
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

	if fh.file.qid.Type&proto9.QTFILE != 0 {
		hostfile, err := os.Open(fh.file.path)
		if err != nil {
			return &proto9.Rerror{
				Tag: msg.Tag,
				Err: err.Error(),
			}
		}
		fh.hostfile = hostfile
	}
	return &proto9.Ropen{
		Tag:    msg.Tag,
		Qid:    fh.file.qid,
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

	if fh.file.qid.Type&proto9.QTDIR != 0 {
		if msg.Offset == 0 {
			dirents, err := ioutil.ReadDir(fh.file.path)
			if err != nil {
				return &proto9.Rerror{
					Tag: msg.Tag,
					Err: err.Error(),
				}
			}
			n := len(dirents)
			fh.stats = make([]proto9.Stat, n, n)
			for i := range dirents {
				fh.stats[i] = srv.dirEntToStat(dirents[i])
			}
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

		if fh.hostfile == nil {
			return &proto9.Rerror{
				Tag: msg.Tag,
				Err: ErrFileNotOpen.Error(),
			}
		}

		n, err := fh.hostfile.ReadAt(buf, int64(msg.Offset))
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
	fh, ok := srv.fids[msg.Fid]
	if !ok {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: ErrNoSuchFid.Error(),
		}
	}

	stat, err := os.Stat(fh.file.path)
	if err != nil {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: err.Error(),
		}
	}

	stat9 := srv.dirEntToStat(stat)
	stat9.Qid.Path = fh.file.qid.Path
	return &proto9.Rstat{
		Tag:  msg.Tag,
		Stat: stat9,
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
