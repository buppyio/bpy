package proto9

import (
	"encoding/binary"
	"errors"
)

var ErrBuffTooSmall = errors.New("buffer too small for message")

type Msg interface {
	MsgType() MessageType
	WireLen() int
	PackBody([]byte)
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
