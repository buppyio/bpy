package main

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/proto9"
	"encoding/binary"
	"encoding/hex"
	"errors"
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
)

type File struct {
	hash   [32]byte
	dirent fs.DirEnt
	qid    proto9.Qid
	// Only valid if qid.Type & QTFILE
	rdr *fs.FileReader
	// Only valid if qid.Type & QTDIR
	dirents fs.DirEnts
	data    []byte
}

func (f *File) Close() error {
	if f.qid.Type&proto9.QTFILE != 0 {
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
	fids           map[proto9.Fid]*File
}

func dirEntToStat(ent *fs.DirEnt) proto9.Stat {
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
		Qid:    proto9.Qid{Type: qtype, Version: 0, Path: 0},
		Length: uint64(ent.Size),
		UID:    "foobar",
		GID:    "foobar",
		MUID:   "foobar",
	}
}

func packDir(dir fs.DirEnts) []byte {
	n := len(dir)
	stats := make([]proto9.Stat, n, n)
	for i := range dir {
		stats[i] = dirEntToStat(&dir[i])
	}
	nbytes := 0
	for i := range stats {
		nbytes += proto9.StatLen(&stats[i])
	}
	buf := make([]byte, nbytes, nbytes)
	offset := 0
	for i := range stats {
		statlen := proto9.StatLen(&stats[i])
		proto9.PackStat(buf[offset:offset+statlen], &stats[i])
		offset += statlen
	}
	return buf
}

func (srv *proto9Server) nextQidPath() uint64 {
	p := srv.qidPathCount
	srv.qidPathCount++
	return p
}

func (srv *proto9Server) makeRoot(hash [32]byte) (*File, error) {
	ents, err := fs.ReadDir(srv.store, hash)
	if err != nil {
		return nil, err
	}
	return &File{
		hash:    hash,
		dirent:  ents[0],
		qid:     proto9.Qid{Type: proto9.QTDIR, Version: 0, Path: 0},
		dirents: ents,
		data:    packDir(ents[1:]),
	}, nil
}

func (srv *proto9Server) AddFid(fid proto9.Fid, f *File) error {
	if fid == proto9.NOFID {
		return ErrBadFid
	}
	_, ok := srv.fids[fid]
	if ok {
		return ErrFidInUse
	}
	srv.fids[fid] = f
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

	err = srv.AddFid(msg.Fid, rootFile)
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
	wf, ok := srv.fids[msg.Fid]
	if !ok {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: ErrNoSuchFid.Error(),
		}
	}
	wqids := make([]proto9.Qid, 0, len(msg.Names))
	whash := wf.hash
	var err error
	var dirents fs.DirEnts
	for i, name := range msg.Names {
		found := false
		if name == "." || name == ".." || name == "" {
			err = ErrBadPath
			goto walkerr
		}
		dirents, err = fs.ReadDir(srv.store, whash)
		if err != nil {
			goto walkerr
		}
		for _, ent := range dirents[1:] {
			if ent.Name == name {
				if !ent.Mode.IsDir() {
					err = ErrNotDir
					goto walkerr
				}
				wqids = append(wqids, proto9.Qid{Type: proto9.QTDIR})
				found = true
				whash = ent.Data
				break
			}
		}
		if !found {
			err = ErrNotExist
			goto walkerr
		}
		continue
	walkerr:
		if i == 0 {
			return &proto9.Rerror{
				Tag: msg.Tag,
				Err: err.Error(),
			}
		}
		break
	}

	if len(wqids) == len(msg.Names) {
		dirents, err := fs.ReadDir(srv.store, whash)
		if err != nil {
			return &proto9.Rerror{
				Tag: msg.Tag,
				Err: err.Error(),
			}
		}
		// XXX handle when newfid == oldfid
		err = srv.AddFid(msg.NewFid,
			&File{
				hash:    whash,
				dirent:  dirents[0],
				qid:     proto9.Qid{Type: proto9.QTDIR, Version: 0, Path: 0},
				dirents: dirents,
				data:    packDir(dirents[1:]),
			})
		if err != nil {
			return &proto9.Rerror{
				Tag: msg.Tag,
				Err: err.Error(),
			}
		}
	}
	return &proto9.Rwalk{
		Tag:  msg.Tag,
		Qids: wqids,
	}
}

func (srv *proto9Server) handleOpen(msg *proto9.Topen) proto9.Msg {
	f, ok := srv.fids[msg.Fid]
	if !ok {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: ErrNoSuchFid.Error(),
		}
	}
	return &proto9.Ropen{
		Tag:    msg.Tag,
		Qid:    f.qid,
		Iounit: srv.negMessageSize - proto9.WriteOverhead,
	}
}

func (srv *proto9Server) handleRead(msg *proto9.Tread) proto9.Msg {
	f, ok := srv.fids[msg.Fid]
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

	if f.dirent.Mode.IsDir() {
		if uint64(len(f.data)) < msg.Offset+nbytes {
			nbytes = uint64(len(f.data)) - msg.Offset
		}
		buf := f.data[msg.Offset : msg.Offset+nbytes]
		return &proto9.Rread{
			Tag:  msg.Tag,
			Data: buf,
		}
	} else {
		return &proto9.Rerror{
			Tag: msg.Tag,
			Err: "unimplemented read",
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

func (srv *proto9Server) serveConn(c net.Conn) {
	defer c.Close()
	srv.fids = make(map[proto9.Fid]*File)
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

func srv9p() {

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
