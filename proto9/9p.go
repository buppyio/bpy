package proto9

import (
	"encoding/binary"
	"errors"
)

var (
	ErrBuffTooSmall = errors.New("buffer too small for message")
	ErrMsgCorrupt   = errors.New("message corrupt")
)

type Msg interface {
	MsgType() MessageType
	WireLen() int
	PackBody([]byte)
	UnpackBody([]byte) error
}

type Qid struct {
	Type    QidType
	Version uint32
	Path    uint64
}

type Stat struct {
	Type   uint16
	Dev    uint32
	Qid    Qid
	Mode   FileMode
	Atime  uint32
	Mtime  uint32
	Length uint64
	Name   string
	UID    string
	GID    string
	MUID   string
}

func NewMsg(mt MessageType) (Msg, error) {
	switch mt {
	case Mt_Tversion:
		return &TVersion{}, nil
	}
	return nil, ErrMsgCorrupt
}

func PackQid(buf []byte, qid Qid) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func PackStat(buf []byte, st Stat) ([]byte, error) {
	return nil, errors.New("unimplemented")
}

func PackMsg(buf []byte, msg Msg) ([]byte, error) {
	nreq := msg.WireLen()
	if len(buf) < nreq {
		return nil, ErrBuffTooSmall
	}
	buf[0] = byte(msg.MsgType())
	binary.LittleEndian.PutUint32(buf[1:HeaderSize], uint32(nreq))
	msg.PackBody(buf[HeaderSize:nreq])
	return buf[0:nreq], nil
}

func UnpackMsg(buf []byte) (Msg, error) {
	if len(buf) < 5 {
		return nil, ErrBuffTooSmall
	}
	msg, err := NewMsg(MessageType(buf[0]))
	if err != nil {
		return nil, err
	}
	err = msg.UnpackBody(buf[5:])
	if err != nil {
		return nil, err
	}
	return msg, nil
}

type TVersion struct {
	Tag         Tag
	MessageSize uint32
	Version     string
}

func (msg *TVersion) MsgType() MessageType {
	return Mt_Tversion
}

func truncstrlen(s string) int {
	return int(uint16(len(s)))
}

func (msg *TVersion) WireLen() int {
	return HeaderSize + 2 + 4 + 2 + truncstrlen(msg.Version)
}

func (msg *TVersion) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.MessageSize))
	strlen := uint16(len(msg.Version))
	binary.LittleEndian.PutUint16(b[6:8], strlen)
	copy(b[8:], []byte(msg.Version)[:strlen])
}

func (msg *TVersion) UnpackBody(b []byte) error {
	sz := 2 + 4 + 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.MessageSize = binary.LittleEndian.Uint32(b[2:6])
	idx := 6
	strlen := int(binary.LittleEndian.Uint16(b[idx : idx+2]))
	sz += strlen
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Version = string(b[idx+2 : idx+2+sz])
	return nil
}
