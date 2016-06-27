package proto

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	TERROR = iota
	TATTACH
	RATTACH
	TNEWTAG
	RNEWTAG
	TREMOVETAG
	RREMOVETAG
	TOPEN
	ROPEN
	TREADAT
	RREADAT
	TNEWPACK
	RNEWPACK
	TWRITEPACK
	RPACKERROR
	TCLOSEPACK
	RCLOSEPACK
)

var (
	ErrMsgTooLarge    = errors.New("Message too large")
	ErrStringTooLarge = errors.New("String too large")
	ErrMsgCorrupt     = errors.New("Message corrupt")
)

type Message interface {
	PackedSize() (uint32, error)
	Pack(buf []byte) error
}

type TError struct {
	Mid     uint16
	Message string
}

type TAttach struct {
	Mid            uint16
	MaxMessageSize uint32
	Version        string
	KeyId          string
}

type RAttach struct {
	Mid            uint16
	MaxMessageSize uint64
}

type TNewTag struct {
	Mid   uint16
	Key   string
	Value string
}

type RNewTag struct {
	Mid uint16
}

type TRemoveTag struct {
	Mid      uint16
	Key      string
	OldValue string
}

type RRemoveTag struct {
	Mid uint16
}

type TOpen struct {
	Mid  uint16
	Fid  uint32
	Path string
}

type ROpen struct {
	Mid  uint16
	Size uint64
}

type TReadAt struct {
	Mid uint16
	Fid uint32
}

type RReadAt struct {
	Mid    uint16
	Offset uint64
	Data   []byte
}

type TNewPack struct {
	Mid uint16
	Pid uint32
}

type RNewPack struct {
	Mid uint16
}

type TWritePack struct {
	Mid  uint16
	Pid  uint32
	Data []byte
}

type RPackError struct {
	Pid     uint32
	Message string
}

type TClosePack struct {
	Mid uint16
	Pid uint32
}

type RClosePack struct {
	Mid uint16
}

func ReadMessage(r io.Reader, buf []byte) (Message, error) {
	_, err := io.ReadFull(r, buf[:4])
	if err != nil {
		return nil, err
	}

	sz := binary.BigEndian.Uint32(buf[:4])

	if sz > uint32(len(buf)) {
		return nil, ErrMsgTooLarge
	}

	_, err = io.ReadFull(r, buf[4:sz])
	if err != nil {
		return nil, err
	}

	return UnpackMessage(buf[:sz])
}

func UnpackMessage(buf []byte) (Message, error) {
	switch buf[4] {
	case TERROR:
		return unpackTError(buf)
	case TATTACH:
		return unpackTAttach(buf)
	case RATTACH:
		return unpackRAttach(buf)
	case TNEWTAG:
		return unpackTNewTag(buf)
	case RNEWTAG:
		return unpackRNewTag(buf)
	case TREMOVETAG:
		return unpackTRemoveTag(buf)
	case RREMOVETAG:
		return unpackRRemoveTag(buf)
	case TOPEN:
		return unpackTOpen(buf)
	case ROPEN:
		return unpackTOpen(buf)
	case TREADAT:
		return unpackTReadAt(buf)
	case RREADAT:
		return unpackRReadAt(buf)
	case TNEWPACK:
		return unpackTNewPack(buf)
	case RNEWPACK:
		return unpackRNewPack(buf)
	case TWRITEPACK:
		return unpackTWritePack(buf)
	case RPACKERROR:
		return unpackRPackError(buf)
	case TCLOSEPACK:
		return unpackTClosePack(buf)
	case RCLOSEPACK:
		return unpackRClosePack(buf)
	default:
		return nil, ErrMsgCorrupt
	}
}

func PackMessage(m Message, buf []byte) (uint32, error) {
	sz, err := m.PackedSize()
	if err != nil {
		return 0, err
	}
	if sz > uint32(len(buf)) {
		return 0, ErrMsgTooLarge
	}
	binary.BigEndian.PutUint32(buf[0:4], sz)
	return sz, m.Pack(buf)
}

func packedStringLen(str string) (uint32, error) {
	if len(str) > 65535 {
		return 0, ErrStringTooLarge
	}
	return 2 + uint32(len(str)), nil
}

func PackString(str string, buf []byte) uint32 {
	n := uint16(copy(buf[2:], []byte(str)))
	binary.BigEndian.PutUint16(buf[0:2], n)
	return 2 + uint32(n)
}

func (m *TError) PackedSize() (uint32, error) {
	msgLen, err := packedStringLen(m.Message)
	if err != nil {
		return 0, err
	}
	return 5 + 2 + msgLen, nil
}

func (m *TError) Pack(buf []byte) error {
	buf[4] = TERROR
	binary.BigEndian.PutUint16(buf[5:7], m.Mid)
	PackString(m.Message, buf[7:])
	return nil
}

func unpackTError(buf []byte) (Message, error) {
	m := &TError{}

	if len(buf) < 5+2+2 {
		return nil, ErrMsgCorrupt
	}
	m.Mid = binary.BigEndian.Uint16(buf[5:7])
	msgLen := uint32(binary.BigEndian.Uint16(buf[7:9]))
	if uint32(len(buf)) < 5+2+2+msgLen {
		return nil, ErrMsgCorrupt
	}
	m.Message = string(buf[9 : 9+msgLen])
	return m, nil
}

func (m *TAttach) PackedSize() (uint32, error) {
	versionLen, err := packedStringLen(m.KeyId)
	if err != nil {
		return 0, err
	}
	keyLen, err := packedStringLen(m.KeyId)
	if err != nil {
		return 0, err
	}
	return 5 + 2 + 4 + 2 + versionLen + 2 + keyLen, nil
}

func (m *TAttach) Pack(buf []byte) error {
	buf[4] = TATTACH
	binary.BigEndian.PutUint16(buf[5:7], m.Mid)
	binary.BigEndian.PutUint32(buf[7:11], m.MaxMessageSize)
	versionLen := PackString(m.Version, buf[11:])
	PackString(m.KeyId, buf[11+versionLen:])
	return nil
}

func unpackTAttach(buf []byte) (Message, error) {
	m := &TAttach{}
	if len(buf) < 5+2+4+2+2 {
		return nil, ErrMsgCorrupt
	}
	m.Mid = binary.BigEndian.Uint16(buf[5:7])
	m.MaxMessageSize = binary.BigEndian.Uint32(buf[7:11])
	versionLen := uint32(binary.BigEndian.Uint16(buf[11:13]))
	if uint32(len(buf)) < 5+2+4+2+2+versionLen {
		return nil, ErrMsgCorrupt
	}
	m.Version = string(buf[13 : 13+versionLen])
	keyLen := uint32(binary.BigEndian.Uint16(buf[13+versionLen : 13+versionLen+2]))
	if uint32(len(buf)) < 5+2+4+2+2+versionLen+keyLen {
		return nil, ErrMsgCorrupt
	}
	m.KeyId = string(buf[13+versionLen+2 : 13+versionLen+2+keyLen])
	return m, nil
}


func (m *RAttach) PackedSize() (uint32, error) {
	return 5 + 2 + 4
}

func (m *RAttach) Pack(buf []byte) error {
	buf[4] = RATTACH
	binary.BigEndian.PutUint16(buf[5:7], m.Mid)
	binary.BigEndian.PutUint32(buf[7:11], m.MaxMessageSize)
	return nil
}

func unpackRAttach(buf []byte) (Message, error) {
	m := &RAttach{}
	m.Mid = binary.BigEndian.Uint16(buf[5:7])
	m.MasMessageSize = binary.BigEndian.Uint32(buf[7:11])
	return m, nil
}

func unpackTNewTag(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackRNewTag(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackTRemoveTag(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackRRemoveTag(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackTOpen(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackROpen(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackTReadAt(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackRReadAt(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackTNewPack(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackRNewPack(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackTWritePack(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackRPackError(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackTClosePack(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}

func unpackRClosePack(buf []byte) (Message, error) {
	return nil, ErrMsgCorrupt
}
