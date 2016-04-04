package proto9

const (
	HeaderSize     = 4 + 1
	ReadOverhead   = HeaderSize + 4
	WriteOverhead  = HeaderSize + 4 + 8 + 4
	QidSize        = 13
	Version        = "9P2000"
	UnknownVersion = "unknown"
)

const (
	Mt_Tversion MessageType = 100 + iota
	Mt_Rversion
	Mt_Tauth
	Mt_Rauth
	Mt_Tattach
	Mt_Rattach
	_
	Mt_Rerror
	Mt_Tflush
	Mt_Rflush
	Mt_Twalk
	Mt_Rwalk
	Mt_Topen
	Mt_Ropen
	Mt_Tcreate
	Mt_Rcreate
	Mt_Tread
	Mt_Rread
	Mt_Twrite
	Mt_Rwrite
	Mt_Tclunk
	Mt_Rclunk
	Mt_Tremove
	Mt_Rremove
	Mt_Tstat
	Mt_Rstat
	Mt_Twstat
	Mt_Rwstat
)

const (
	NOTAG Tag = 0xFFFF
	NOFID Fid = 0xFFFFFFFF
)

const (
	OREAD OpenMode = iota
	OWRITE
	ORDWR
	OEXEC

	OTRUNC OpenMode = 16 * (iota + 1)
	OCEXEC
	ORCLOSE
)

const (
	DMDIR    FileMode = 0x80000000
	DMAPPEND FileMode = 0x40000000
	DMEXCL   FileMode = 0x20000000
	DMMOUNT  FileMode = 0x10000000
	DMAUTH   FileMode = 0x08000000
	DMTMP    FileMode = 0x04000000
	DMREAD   FileMode = 0x4
	DMWRITE  FileMode = 0x2
	DMEXEC   FileMode = 0x1
)

const (
	QTFILE   QidType = 0x00
	QTTMP    QidType = 0x04
	QTAUTH   QidType = 0x08
	QTMOUNT  QidType = 0x10
	QTEXCL   QidType = 0x20
	QTAPPEND QidType = 0x40
	QTDIR    QidType = 0x80
)

type MessageType byte
type Tag uint16
type Fid uint32
type FileMode uint32
type OpenMode byte
type QidType byte
