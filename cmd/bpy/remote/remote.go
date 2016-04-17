package remote

import (
	"acha.ninja/bpy/proto9"
	"encoding/binary"
	"errors"
	"io/ioutil"
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
)

type File struct {
	path string
	qid  proto9.Qid
}

type FileHandle struct {
	file *File

	osfile *os.File

	diroffset uint64
	stats     []proto9.Stat
}

func (f *FileHandle) Close() error {
	if f.osfile != nil {
		err := f.osfile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

type proto9Server struct {
	root           string
	maxMessageSize uint32
	negMessageSize uint32
	inbuf          []byte
	outbuf         []byte
	qidPathCount   uint64
	fids           map[proto9.Fid]*FileHandle
}

func (srv *proto9Server) StatRoot() ([]proto9.Stat, error) {
	dirents, err := ioutil.ReadDir(srv.root)
	if err != nil {
		return nil, err
	}
	stats := make([]proto9.Stat, 0, len(dirents))
	for _, ent := range dirents {
		if ent.Mode().IsDir() {
			continue
		}
		stat := proto9.Stat{
			Name:   ent.Name(),
			Mode:   0666,
			Length: uint64(ent.Size()),
			Qid: proto9.Qid{Type: proto9.QTFILE,
				Path:    srv.qidPathCount,
				Version: uint32(ent.ModTime().UnixNano() / 1000000),
			},
		}
		srv.qidPathCount++
		stats = append(stats, stat)
	}
	return stats, nil
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
	rootFile := &File{
		path: srv.root,
	}
	f := &FileHandle{
		file: rootFile,
	}
	err := srv.AddFid(msg.Fid, f)
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
	path := "/home/ac/.bpy/store"
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
			root:           path,
			maxMessageSize: 4096,
		}
		go srv.serveConn(c)
	}
}
