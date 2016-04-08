package main

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/proto9"
	"encoding/binary"
	"errors"
	"log"
	"net"
)

var (
	ErrNoSuchFid = errors.New("no such fid")
	ErrFidInUse  = errors.New("fid in use")
)

type File struct {
	hash   [32]byte
	dirent fs.DirEnt
	qid    proto9.Qid
	// Only valid if qid.Type & QTFILE
	rdr *fs.Reader
	// Only valid if qid.Type & QTDIR
	data []byte
}

type proto9Server struct {
	maxMessageSize uint32
	negMessageSize uint32
	inbuf          []byte
	outbuf         []byte
	qidPathCount   uint64
	fids           map[Fid]*File
}

func (srv *proto9Server) nextQidPath() {
	p := qidPathCount
	qidPathCount++
	return p
}

func (srv *proto9Server) AddFid(fid Fid, f *File) error {
	_, ok := srv.fids[fid]
	if !ok {
		return ErrFidInUse
	}
	srv.fids[fid] = f
	return nil
}

func (srv *proto9Server) ClunkFid(fid Fid) error {
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
			Err: "version tag incorrect",
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
			Err: "auth not supported",
		}
	}

	return &proto9.Rattach{
		Tag: msg.Tag,
	}
}

func (srv *proto9Server) serveConn(c net.Conn) {
	defer c.Close()
	srv.inbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	srv.outbuf = make([]byte, srv.maxMessageSize, srv.maxMessageSize)
	for {
		var resp Msg
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
		err = srv.sendMsg(c, resp)
		if err != nil {
			log.Printf("error sending message: %s", err.Error())
			return
		}
	}

}

func srv9p() {
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
			maxMessageSize: 4096,
		}
		go srv.serveConn(c)
	}
}
