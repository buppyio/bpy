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
		return &Tversion{}, nil
	case Mt_Rversion:
		return &Rversion{}, nil
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
	binary.LittleEndian.PutUint32(buf[0:4], uint32(nreq))
	buf[4] = byte(msg.MsgType())
	msg.PackBody(buf[HeaderSize:nreq])
	return buf[0:nreq], nil
}

func UnpackMsg(buf []byte) (Msg, error) {
	if len(buf) < 5 {
		return nil, ErrBuffTooSmall
	}
	msg, err := NewMsg(MessageType(buf[4]))
	if err != nil {
		return nil, err
	}
	err = msg.UnpackBody(buf[5:])
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func truncstrlen(s string) int {
	return int(uint16(len(s)))
}

type Tversion struct {
	Tag         Tag
	MessageSize uint32
	Version     string
}

func (msg *Tversion) MsgType() MessageType {
	return Mt_Tversion
}

func (msg *Tversion) WireLen() int {
	return HeaderSize + 2 + 4 + 2 + truncstrlen(msg.Version)
}

func (msg *Tversion) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.MessageSize))
	strlen := uint16(len(msg.Version))
	binary.LittleEndian.PutUint16(b[6:8], strlen)
	copy(b[8:], []byte(msg.Version)[:strlen])
}

func (msg *Tversion) UnpackBody(b []byte) error {
	sz := 2 + 4 + 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.MessageSize = binary.LittleEndian.Uint32(b[2:6])
	strlen := int(binary.LittleEndian.Uint16(b[6:8]))
	sz += strlen
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Version = string(b[8 : 8+strlen])
	return nil
}

type Rversion struct {
	Tag         Tag
	MessageSize uint32
	Version     string
}

func (msg *Rversion) MsgType() MessageType {
	return Mt_Rversion
}

func (msg *Rversion) WireLen() int {
	return HeaderSize + 2 + 4 + 2 + truncstrlen(msg.Version)
}

func (msg *Rversion) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.MessageSize))
	strlen := uint16(len(msg.Version))
	binary.LittleEndian.PutUint16(b[6:8], strlen)
	copy(b[8:], []byte(msg.Version)[:strlen])
}

func (msg *Rversion) UnpackBody(b []byte) error {
	sz := 2 + 4 + 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.MessageSize = binary.LittleEndian.Uint32(b[2:6])
	strlen := int(binary.LittleEndian.Uint16(b[6:8]))
	sz += strlen
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Version = string(b[8 : 8+strlen])
	return nil
}
