package proto9

import (
	"encoding/binary"
	"errors"
	"io"
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

func (q *Qid) IsDir() bool {
	return q.Type == QTDIR
}

func (q *Qid) IsFile() bool {
	return q.Type == QTFILE
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
	case Mt_Tattach:
		return &Tattach{}, nil
	case Mt_Rattach:
		return &Rattach{}, nil
	case Mt_Tauth:
		return &Tauth{}, nil
	case Mt_Rauth:
		return &Rauth{}, nil
	case Mt_Rerror:
		return &Rerror{}, nil
	case Mt_Tflush:
		return &Tflush{}, nil
	case Mt_Rflush:
		return &Rflush{}, nil
	case Mt_Tread:
		return &Tread{}, nil
	case Mt_Rread:
		return &Rread{}, nil
	case Mt_Twrite:
		return &Twrite{}, nil
	case Mt_Rwrite:
		return &Rwrite{}, nil
	case Mt_Tclunk:
		return &Tclunk{}, nil
	case Mt_Rclunk:
		return &Rclunk{}, nil
	case Mt_Tremove:
		return &Tremove{}, nil
	case Mt_Rremove:
		return &Rremove{}, nil
	case Mt_Topen:
		return &Topen{}, nil
	case Mt_Ropen:
		return &Ropen{}, nil
	case Mt_Tcreate:
		return &Tcreate{}, nil
	case Mt_Rcreate:
		return &Rcreate{}, nil
	case Mt_Tstat:
		return &Tstat{}, nil
	case Mt_Rstat:
		return &Rstat{}, nil
	case Mt_Twstat:
		return &Twstat{}, nil
	case Mt_Rwstat:
		return &Rwstat{}, nil
	case Mt_Twalk:
		return &Twalk{}, nil
	case Mt_Rwalk:
		return &Rwalk{}, nil
	}
	return nil, ErrMsgCorrupt
}

func PackQid(buf []byte, qid Qid) {
	buf[0] = byte(qid.Type)
	binary.LittleEndian.PutUint32(buf[1:5], qid.Version)
	binary.LittleEndian.PutUint64(buf[5:13], qid.Path)
}

func UnpackQid(buf []byte, qid *Qid) {
	qid.Type = QidType(buf[0])
	qid.Version = binary.LittleEndian.Uint32(buf[1:5])
	qid.Path = binary.LittleEndian.Uint64(buf[5:13])
}

func StatLen(st *Stat) int {
	return 2 + 2 + 4 + QidSize + 4 + 4 + 4 + 8 + 2 + truncstrlen(st.Name) + 2 + truncstrlen(st.UID) + 2 + truncstrlen(st.GID) + 2 + truncstrlen(st.MUID)
}

func PackStat(buf []byte, st *Stat) {
	binary.LittleEndian.PutUint16(buf[0:2], uint16(StatLen(st)-2))
	binary.LittleEndian.PutUint16(buf[2:4], st.Type)
	binary.LittleEndian.PutUint32(buf[4:8], st.Dev)
	PackQid(buf[8:21], st.Qid)
	binary.LittleEndian.PutUint32(buf[21:25], uint32(st.Mode))
	binary.LittleEndian.PutUint32(buf[25:29], st.Atime)
	binary.LittleEndian.PutUint32(buf[29:33], st.Mtime)
	binary.LittleEndian.PutUint64(buf[33:41], st.Length)
	namelen := truncstrlen(st.Name)
	uidlen := truncstrlen(st.UID)
	gidlen := truncstrlen(st.GID)
	muidlen := truncstrlen(st.MUID)
	binary.LittleEndian.PutUint16(buf[41:], uint16(namelen))
	binary.LittleEndian.PutUint16(buf[43+namelen:], uint16(uidlen))
	binary.LittleEndian.PutUint16(buf[45+namelen+uidlen:], uint16(gidlen))
	binary.LittleEndian.PutUint16(buf[47+namelen+uidlen+gidlen:], uint16(muidlen))
	copy(buf[43:43+namelen], st.Name)
	copy(buf[45+namelen:45+namelen+uidlen], st.UID)
	copy(buf[47+namelen+uidlen:47+namelen+uidlen+gidlen], st.GID)
	copy(buf[49+namelen+uidlen+gidlen:49+namelen+uidlen+gidlen+muidlen], st.MUID)
}

func UnpackStat(buf []byte, st *Stat) (int, error) {
	sz := 2 + 2 + 4 + QidSize + 4 + 4 + 4 + 8 + 2 + 2 + 2 + 2
	if len(buf) < sz {
		return 0, ErrMsgCorrupt
	}
	st.Type = binary.LittleEndian.Uint16(buf[2:4])
	st.Dev = binary.LittleEndian.Uint32(buf[4:8])
	UnpackQid(buf[8:21], &st.Qid)
	st.Mode = FileMode(binary.LittleEndian.Uint32(buf[21:25]))
	st.Atime = binary.LittleEndian.Uint32(buf[25:29])
	st.Mtime = binary.LittleEndian.Uint32(buf[29:33])
	st.Length = binary.LittleEndian.Uint64(buf[33:41])
	namelen := int(binary.LittleEndian.Uint16(buf[41:43]))
	sz += namelen
	if len(buf) < sz {
		return 0, ErrMsgCorrupt
	}
	uidlen := int(binary.LittleEndian.Uint16(buf[43+namelen : 45+namelen]))
	sz += uidlen
	if len(buf) < sz {
		return 0, ErrMsgCorrupt
	}
	gidlen := int(binary.LittleEndian.Uint16(buf[45+namelen+uidlen : 47+namelen+uidlen]))
	sz += gidlen
	if len(buf) < sz {
		return 0, ErrMsgCorrupt
	}
	muidlen := int(binary.LittleEndian.Uint16(buf[47+namelen+uidlen+gidlen : 49+namelen+uidlen+gidlen]))
	sz += muidlen
	if len(buf) < sz {
		return 0, ErrMsgCorrupt
	}
	st.Name = string(buf[43 : 43+namelen])
	st.UID = string(buf[45+namelen : 45+namelen+uidlen])
	st.GID = string(buf[47+namelen+uidlen : 47+namelen+uidlen+gidlen])
	st.MUID = string(buf[49+namelen+uidlen+gidlen : 49+namelen+uidlen+gidlen+muidlen])
	return sz, nil
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

type Tauth struct {
	Tag   Tag
	Afid  Fid
	Uname string
	Aname string
}

func (msg *Tauth) MsgType() MessageType {
	return Mt_Tauth
}

func (msg *Tauth) WireLen() int {
	return HeaderSize + 2 + 4 + 2 + truncstrlen(msg.Uname) + 2 + truncstrlen(msg.Aname)
}

func (msg *Tauth) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Afid))
	unamelen := uint16(len(msg.Uname))
	binary.LittleEndian.PutUint16(b[6:8], unamelen)
	copy(b[8:], []byte(msg.Uname)[:unamelen])
	anamelen := uint16(len(msg.Aname))
	binary.LittleEndian.PutUint16(b[8+unamelen:10+unamelen], anamelen)
	copy(b[10+unamelen:], []byte(msg.Aname)[:anamelen])
}

func (msg *Tauth) UnpackBody(b []byte) error {
	sz := 2 + 4 + 2 + 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Afid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	unamelen := int(binary.LittleEndian.Uint16(b[6:8]))
	sz += unamelen
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Uname = string(b[8 : 8+unamelen])
	anamelen := int(binary.LittleEndian.Uint16(b[8+unamelen : 10+unamelen]))
	sz += anamelen
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Aname = string(b[10+unamelen : 10+unamelen+anamelen])
	return nil
}

type Rauth struct {
	Tag  Tag
	Aqid Qid
}

func (msg *Rauth) MsgType() MessageType {
	return Mt_Rauth
}

func (msg *Rauth) WireLen() int {
	return HeaderSize + 2 + QidSize
}

func (msg *Rauth) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	PackQid(b[2:], msg.Aqid)
}

func (msg *Rauth) UnpackBody(b []byte) error {
	sz := 2 + QidSize
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	UnpackQid(b[2:QidSize], &msg.Aqid)
	return nil
}

type Tattach struct {
	Tag   Tag
	Fid   Fid
	Afid  Fid
	Uname string
	Aname string
}

func (msg *Tattach) MsgType() MessageType {
	return Mt_Tattach
}

func (msg *Tattach) WireLen() int {
	return HeaderSize + 2 + 4 + 4 + +2 + truncstrlen(msg.Uname) + 2 + truncstrlen(msg.Aname)
}

func (msg *Tattach) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Fid))
	binary.LittleEndian.PutUint32(b[6:10], uint32(msg.Afid))
	unamelen := uint16(len(msg.Uname))
	binary.LittleEndian.PutUint16(b[10:12], unamelen)
	copy(b[12:], []byte(msg.Uname)[:unamelen])
	anamelen := uint16(len(msg.Aname))
	binary.LittleEndian.PutUint16(b[12+unamelen:14+unamelen], anamelen)
	copy(b[14+unamelen:], []byte(msg.Aname)[:anamelen])
}

func (msg *Tattach) UnpackBody(b []byte) error {
	sz := 2 + 4 + 4 + 2 + 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Fid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	msg.Afid = Fid(binary.LittleEndian.Uint32(b[6:10]))
	unamelen := int(binary.LittleEndian.Uint16(b[10:12]))
	sz += unamelen
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Uname = string(b[12 : 12+unamelen])
	anamelen := int(binary.LittleEndian.Uint16(b[12+unamelen : 14+unamelen]))
	sz += anamelen
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Aname = string(b[14+unamelen : 14+unamelen+anamelen])
	return nil
}

type Rattach struct {
	Tag Tag
	Qid Qid
}

func (msg *Rattach) MsgType() MessageType {
	return Mt_Rattach
}

func (msg *Rattach) WireLen() int {
	return HeaderSize + 2 + QidSize
}

func (msg *Rattach) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	PackQid(b[2:], msg.Qid)
}

func (msg *Rattach) UnpackBody(b []byte) error {
	sz := 2 + QidSize
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	UnpackQid(b[2:QidSize], &msg.Qid)
	return nil
}

type Rerror struct {
	Tag Tag
	Err string
}

func (msg *Rerror) MsgType() MessageType {
	return Mt_Rerror
}

func (msg *Rerror) WireLen() int {
	return HeaderSize + 2 + 2 + truncstrlen(msg.Err)
}

func (msg *Rerror) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	errlen := uint16(len(msg.Err))
	binary.LittleEndian.PutUint16(b[2:4], errlen)
	copy(b[4:], []byte(msg.Err)[:errlen])
}

func (msg *Rerror) UnpackBody(b []byte) error {
	sz := 2 + 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	errlen := int(binary.LittleEndian.Uint16(b[2:4]))
	sz += errlen
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Err = string(b[4 : 4+errlen])
	return nil
}

type Tflush struct {
	Tag    Tag
	OldTag Tag
}

func (msg *Tflush) MsgType() MessageType {
	return Mt_Tflush
}

func (msg *Tflush) WireLen() int {
	return HeaderSize + 2 + 2
}

func (msg *Tflush) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint16(b[2:4], uint16(msg.OldTag))
}

func (msg *Tflush) UnpackBody(b []byte) error {
	sz := 2 + 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.OldTag = Tag(binary.LittleEndian.Uint16(b[2:4]))
	return nil
}

type Rflush struct {
	Tag Tag
}

func (msg *Rflush) MsgType() MessageType {
	return Mt_Rflush
}

func (msg *Rflush) WireLen() int {
	return HeaderSize + 2
}

func (msg *Rflush) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
}

func (msg *Rflush) UnpackBody(b []byte) error {
	sz := 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	return nil
}

type Tread struct {
	Tag    Tag
	Fid    Fid
	Offset uint64
	Count  uint32
}

func (msg *Tread) MsgType() MessageType {
	return Mt_Tread
}

func (msg *Tread) WireLen() int {
	return HeaderSize + 2 + 4 + 8 + 4
}

func (msg *Tread) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Fid))
	binary.LittleEndian.PutUint64(b[6:14], msg.Offset)
	binary.LittleEndian.PutUint32(b[14:18], msg.Count)
}

func (msg *Tread) UnpackBody(b []byte) error {
	sz := 2 + 4 + 8 + 4
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Fid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	msg.Offset = binary.LittleEndian.Uint64(b[6:14])
	msg.Count = binary.LittleEndian.Uint32(b[14:18])
	return nil
}

type Rread struct {
	Tag  Tag
	Data []byte
}

func (msg *Rread) MsgType() MessageType {
	return Mt_Rread
}

func (msg *Rread) WireLen() int {
	return HeaderSize + 2 + 4 + len(msg.Data)
}

func (msg *Rread) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(len(msg.Data)))
	copy(b[6:], msg.Data)
}

func (msg *Rread) UnpackBody(b []byte) error {
	sz := 2 + 4
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	datalen := binary.LittleEndian.Uint32(b[2:6])
	sz += int(datalen)
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Data = make([]byte, datalen, datalen)
	copy(msg.Data, b[6:6+datalen])
	return nil
}

type Twrite struct {
	Tag    Tag
	Fid    Fid
	Offset uint64
	Data   []byte
}

func (msg *Twrite) MsgType() MessageType {
	return Mt_Twrite
}

func (msg *Twrite) WireLen() int {
	return HeaderSize + 2 + 4 + 8 + 4 + len(msg.Data)
}

func (msg *Twrite) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Fid))
	binary.LittleEndian.PutUint64(b[6:14], uint64(msg.Offset))
	binary.LittleEndian.PutUint32(b[14:18], uint32(len(msg.Data)))
	copy(b[18:], msg.Data)
}

func (msg *Twrite) UnpackBody(b []byte) error {
	sz := 2 + 4 + 8 + 4
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Fid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	msg.Offset = binary.LittleEndian.Uint64(b[6:14])
	datalen := binary.LittleEndian.Uint32(b[14:18])
	sz += int(datalen)
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Data = make([]byte, datalen, datalen)
	copy(msg.Data, b[18:18+datalen])
	return nil
}

type Rwrite struct {
	Tag   Tag
	Count uint32
}

func (msg *Rwrite) MsgType() MessageType {
	return Mt_Rwrite
}

func (msg *Rwrite) WireLen() int {
	return HeaderSize + 2 + 4
}

func (msg *Rwrite) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], msg.Count)
}

func (msg *Rwrite) UnpackBody(b []byte) error {
	sz := 2 + 4
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Count = binary.LittleEndian.Uint32(b[2:6])
	return nil
}

type Tclunk struct {
	Tag Tag
	Fid Fid
}

func (msg *Tclunk) MsgType() MessageType {
	return Mt_Tclunk
}

func (msg *Tclunk) WireLen() int {
	return HeaderSize + 2 + 4
}

func (msg *Tclunk) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Fid))
}

func (msg *Tclunk) UnpackBody(b []byte) error {
	sz := 2 + 4
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Fid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	return nil
}

type Rclunk struct {
	Tag Tag
}

func (msg *Rclunk) MsgType() MessageType {
	return Mt_Rclunk
}

func (msg *Rclunk) WireLen() int {
	return HeaderSize + 2
}

func (msg *Rclunk) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
}

func (msg *Rclunk) UnpackBody(b []byte) error {
	sz := 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	return nil
}

type Tremove struct {
	Tag Tag
	Fid Fid
}

func (msg *Tremove) MsgType() MessageType {
	return Mt_Tremove
}

func (msg *Tremove) WireLen() int {
	return HeaderSize + 2 + 4
}

func (msg *Tremove) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Fid))
}

func (msg *Tremove) UnpackBody(b []byte) error {
	sz := 2 + 4
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Fid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	return nil
}

type Rremove struct {
	Tag Tag
}

func (msg *Rremove) MsgType() MessageType {
	return Mt_Rremove
}

func (msg *Rremove) WireLen() int {
	return HeaderSize + 2
}

func (msg *Rremove) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
}

func (msg *Rremove) UnpackBody(b []byte) error {
	sz := 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	return nil
}

type Topen struct {
	Tag  Tag
	Fid  Fid
	Mode OpenMode
}

func (msg *Topen) MsgType() MessageType {
	return Mt_Topen
}

func (msg *Topen) WireLen() int {
	return HeaderSize + 2 + 4 + 1
}

func (msg *Topen) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Fid))
	b[6] = byte(msg.Mode)
}

func (msg *Topen) UnpackBody(b []byte) error {
	sz := 2 + 4 + 1
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Fid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	msg.Mode = OpenMode(b[6])
	return nil
}

type Ropen struct {
	Tag    Tag
	Qid    Qid
	Iounit uint32
}

func (msg *Ropen) MsgType() MessageType {
	return Mt_Ropen
}

func (msg *Ropen) WireLen() int {
	return HeaderSize + 2 + QidSize + 4
}

func (msg *Ropen) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	PackQid(b[2:15], msg.Qid)
	binary.LittleEndian.PutUint32(b[15:19], msg.Iounit)
}

func (msg *Ropen) UnpackBody(b []byte) error {
	sz := 2 + QidSize + 4
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	UnpackQid(b[2:15], &msg.Qid)
	msg.Iounit = binary.LittleEndian.Uint32(b[15:19])
	return nil
}

type Tcreate struct {
	Tag  Tag
	Fid  Fid
	Name string
	Perm FileMode
	Mode OpenMode
}

func (msg *Tcreate) MsgType() MessageType {
	return Mt_Tcreate
}

func (msg *Tcreate) WireLen() int {
	return HeaderSize + 2 + 4 + 2 + truncstrlen(msg.Name) + 4 + 1
}

func (msg *Tcreate) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Fid))
	namelen := uint16(len(msg.Name))
	binary.LittleEndian.PutUint16(b[6:8], namelen)
	copy(b[8:8+namelen], []byte(msg.Name))
	binary.LittleEndian.PutUint32(b[8+namelen:12+namelen], uint32(msg.Perm))
	b[12+namelen] = byte(msg.Mode)
}

func (msg *Tcreate) UnpackBody(b []byte) error {
	sz := 2 + 4 + 2 + 4 + 1
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Fid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	namelen := binary.LittleEndian.Uint16(b[6:8])
	sz += int(namelen)
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Name = string(b[8 : 8+namelen])
	msg.Perm = FileMode(binary.LittleEndian.Uint32(b[8+namelen : 12+namelen]))
	msg.Mode = OpenMode(b[12+namelen])
	return nil
}

type Rcreate struct {
	Tag    Tag
	Qid    Qid
	Iounit uint32
}

func (msg *Rcreate) MsgType() MessageType {
	return Mt_Rcreate
}

func (msg *Rcreate) WireLen() int {
	return HeaderSize + 2 + QidSize + 4
}

func (msg *Rcreate) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	PackQid(b[2:15], msg.Qid)
	binary.LittleEndian.PutUint32(b[15:19], msg.Iounit)
}

func (msg *Rcreate) UnpackBody(b []byte) error {
	sz := 2 + QidSize + 4
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	UnpackQid(b[2:15], &msg.Qid)
	msg.Iounit = binary.LittleEndian.Uint32(b[15:19])
	return nil
}

type Tstat struct {
	Tag Tag
	Fid Fid
}

func (msg *Tstat) MsgType() MessageType {
	return Mt_Tstat
}

func (msg *Tstat) WireLen() int {
	return HeaderSize + 2 + 4
}

func (msg *Tstat) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Fid))
}

func (msg *Tstat) UnpackBody(b []byte) error {
	sz := 2 + 4
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Fid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	return nil
}

type Rstat struct {
	Tag  Tag
	Stat Stat
}

func (msg *Rstat) MsgType() MessageType {
	return Mt_Rstat
}

func (msg *Rstat) WireLen() int {
	return HeaderSize + 2 + 2 + StatLen(&msg.Stat)
}

func (msg *Rstat) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint16(b[2:4], uint16(StatLen(&msg.Stat)))
	PackStat(b[4:], &msg.Stat)
}

func (msg *Rstat) UnpackBody(b []byte) error {
	sz := 4
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	_, err := UnpackStat(b[4:], &msg.Stat)
	return err
}

type Twstat struct {
	Tag  Tag
	Fid  Fid
	Stat Stat
}

func (msg *Twstat) MsgType() MessageType {
	return Mt_Twstat
}

func (msg *Twstat) WireLen() int {
	return HeaderSize + 2 + 4 + 2 + StatLen(&msg.Stat)
}

func (msg *Twstat) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Fid))
	binary.LittleEndian.PutUint16(b[6:8], uint16(StatLen(&msg.Stat)))
	PackStat(b[8:], &msg.Stat)
}

func (msg *Twstat) UnpackBody(b []byte) error {
	sz := 8
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Fid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	_, err := UnpackStat(b[8:], &msg.Stat)
	return err
}

type Rwstat struct {
	Tag Tag
}

func (msg *Rwstat) MsgType() MessageType {
	return Mt_Rwstat
}

func (msg *Rwstat) WireLen() int {
	return HeaderSize + 2
}

func (msg *Rwstat) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
}

func (msg *Rwstat) UnpackBody(b []byte) error {
	sz := 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	return nil
}

type Twalk struct {
	Tag    Tag
	Fid    Fid
	NewFid Fid
	Names  []string
}

func (msg *Twalk) MsgType() MessageType {
	return Mt_Twalk
}

func (msg *Twalk) WireLen() int {
	sz := HeaderSize + 2 + 4 + 4 + 2
	for _, s := range msg.Names {
		sz += 2 + truncstrlen(s)
	}
	return sz
}

func (msg *Twalk) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint32(b[2:6], uint32(msg.Fid))
	binary.LittleEndian.PutUint32(b[6:10], uint32(msg.NewFid))
	binary.LittleEndian.PutUint16(b[10:12], uint16(len(msg.Names)))
	offset := 0
	for _, n := range msg.Names {
		l := truncstrlen(n)
		binary.LittleEndian.PutUint16(b[12+offset:14+offset], uint16(l))
		copy(b[14+offset:14+offset+l], n[0:l])
		offset += 2 + l
	}
}

func (msg *Twalk) UnpackBody(b []byte) error {
	sz := 2 + 4 + 4 + 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	msg.Fid = Fid(binary.LittleEndian.Uint32(b[2:6]))
	msg.NewFid = Fid(binary.LittleEndian.Uint32(b[6:10]))
	n := int(binary.LittleEndian.Uint16(b[10:12]))
	msg.Names = make([]string, n, n)
	offset := 0
	for i := 0; i < n; i++ {
		sz += 2
		if len(b) < sz {
			return ErrMsgCorrupt
		}
		nlen := int(binary.LittleEndian.Uint16(b[12+offset : 14+offset]))
		msg.Names[i] = string(b[14+offset : 14+offset+nlen])
		offset += int(nlen) + 2
	}
	return nil
}

type Rwalk struct {
	Tag  Tag
	Qids []Qid
}

func (msg *Rwalk) MsgType() MessageType {
	return Mt_Rwalk
}

func (msg *Rwalk) WireLen() int {
	return HeaderSize + 2 + 2 + len(msg.Qids)*QidSize
}

func (msg *Rwalk) PackBody(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], uint16(msg.Tag))
	binary.LittleEndian.PutUint16(b[2:4], uint16(len(msg.Qids)))
	offset := 0
	for i := range msg.Qids {
		PackQid(b[4+offset:4+offset+QidSize], msg.Qids[i])
		offset += QidSize
	}
}

func (msg *Rwalk) UnpackBody(b []byte) error {
	sz := 2 + 2
	if len(b) < sz {
		return ErrMsgCorrupt
	}
	msg.Tag = Tag(binary.LittleEndian.Uint16(b[0:2]))
	n := int(binary.LittleEndian.Uint16(b[2:4]))
	if len(b) < n*QidSize {
		return ErrMsgCorrupt
	}
	msg.Qids = make([]Qid, n, n)
	offset := 0
	for i := 0; i < n; i++ {
		UnpackQid(b[4+offset:4+offset+QidSize], &msg.Qids[i])
		offset += QidSize
	}
	return nil
}

func ReadMsg(r io.Reader, buf []byte) (Msg, error) {
	if len(buf) < 5 {
		return nil, ErrBuffTooSmall
	}
	_, err := io.ReadFull(r, buf[0:5])
	if err != nil {
		return nil, err
	}
	sz := binary.LittleEndian.Uint32(buf[0:4])
	if sz < 5 {
		return nil, ErrMsgCorrupt
	}
	if uint32(len(buf)) < sz {
		return nil, ErrBuffTooSmall
	}
	_, err = io.ReadFull(r, buf[5:sz])
	if err != nil {
		return nil, err
	}
	return UnpackMsg(buf[0:sz])
}

func WriteMsg(w io.Writer, buf []byte, msg Msg) error {
	packed, err := PackMsg(buf, msg)
	if err != nil {
		return err
	}
	_, err = w.Write(packed)
	if err != nil {
		return err
	}
	return nil
}

func MakeError(t Tag, err error) Msg {
	return &Rerror{
		Tag: t,
		Err: err.Error(),
	}
}
