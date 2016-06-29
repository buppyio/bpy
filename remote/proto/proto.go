package proto

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
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
	ErrMsgTooLarge = errors.New("message too large")
	ErrStrTooLarge = errors.New("string too large")
	ErrMsgCorrupt  = errors.New("message corrupt")
)

type Message interface {
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
	Mid    uint16
	Fid    uint32
	Offset uint64
}

type RReadAt struct {
	Mid  uint16
	Data []byte
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
		m := &TError{}
		return m, unpackFields(m, buf[5:])
	case TATTACH:
		m := &TAttach{}
		return m, unpackFields(m, buf[5:])
	case RATTACH:
		m := &RAttach{}
		return m, unpackFields(m, buf[5:])
	case TNEWTAG:
		m := &TNewTag{}
		return m, unpackFields(m, buf[5:])
	case RNEWTAG:
		m := &RNewTag{}
		return m, unpackFields(m, buf[5:])
	case TREMOVETAG:
		m := &TRemoveTag{}
		return m, unpackFields(m, buf[5:])
	case RREMOVETAG:
		m := &RRemoveTag{}
		return m, unpackFields(m, buf[5:])
	case TOPEN:
		m := &TOpen{}
		return m, unpackFields(m, buf[5:])
	case ROPEN:
		m := &ROpen{}
		return m, unpackFields(m, buf[5:])
	case TREADAT:
		m := &TReadAt{}
		return m, unpackFields(m, buf[5:])
	case RREADAT:
		m := &RReadAt{}
		return m, unpackFields(m, buf[5:])
	case TNEWPACK:
		m := &TNewPack{}
		return m, unpackFields(m, buf[5:])
	case RNEWPACK:
		m := &RNewPack{}
		return m, unpackFields(m, buf[5:])
	case TWRITEPACK:
		m := &TWritePack{}
		return m, unpackFields(m, buf[5:])
	case RPACKERROR:
		m := &RPackError{}
		return m, unpackFields(m, buf[5:])
	case TCLOSEPACK:
		m := &TClosePack{}
		return m, unpackFields(m, buf[5:])
	case RCLOSEPACK:
		m := &RClosePack{}
		return m, unpackFields(m, buf[5:])
	default:
		return nil, ErrMsgCorrupt
	}
}

func unpackFields(m Message, buf []byte) error {
	v := reflect.Indirect(reflect.ValueOf(m))
	for i := 0; i < v.NumField(); i++ {
		v := v.Field(i)
		switch v.Kind() {
		case reflect.Uint16:
			if len(buf) < 2 {
				return ErrMsgCorrupt
			}
			v.SetUint(uint64(binary.BigEndian.Uint16(buf[0:2])))
			buf = buf[2:]
		case reflect.Uint32:
			if len(buf) < 4 {
				return ErrMsgCorrupt
			}
			v.SetUint(uint64(binary.BigEndian.Uint32(buf[0:4])))
			buf = buf[4:]
		case reflect.Uint64:
			if len(buf) < 8 {
				return ErrMsgCorrupt
			}
			v.SetUint(binary.BigEndian.Uint64(buf[0:8]))
			buf = buf[8:]
		case reflect.String:
			if len(buf) < 2 {
				return ErrMsgCorrupt
			}
			sz := int(binary.BigEndian.Uint16(buf[0:2]))
			buf = buf[2:]
			if len(buf) < sz {
				return ErrMsgCorrupt
			}
			v.SetString(string(buf[0:sz]))
			buf = buf[sz:]
		case reflect.Slice:
			if len(buf) < 4 {
				return ErrMsgCorrupt
			}
			sz := int(binary.BigEndian.Uint32(buf[0:4]))
			buf = buf[4:]
			if len(buf) < sz {
				return ErrMsgCorrupt
			}
			v.SetBytes(buf[0:sz])
			buf = buf[sz:]
		default:
			panic("internal error")
		}
	}
	if len(buf) != 0 {
		return ErrMsgCorrupt
	}
	return nil
}

func GetMessageType(m Message) byte {
	switch m.(type) {
	case *TError:
		return TERROR
	case *TAttach:
		return TATTACH
	case *RAttach:
		return RATTACH
	case *TNewTag:
		return TNEWTAG
	case *RNewTag:
		return RNEWTAG
	case *TRemoveTag:
		return TREMOVETAG
	case *RRemoveTag:
		return RREMOVETAG
	case *TOpen:
		return TOPEN
	case *ROpen:
		return ROPEN
	case *TReadAt:
		return TREADAT
	case *RReadAt:
		return RREADAT
	case *TNewPack:
		return TNEWPACK
	case *RNewPack:
		return RNEWPACK
	case *TWritePack:
		return TWRITEPACK
	case *RPackError:
		return RPACKERROR
	case *TClosePack:
		return TCLOSEPACK
	case *RClosePack:
		return RCLOSEPACK
	}
	panic("internal error")
}

func PackMessage(m Message, buf []byte) (int, error) {
	origbuf := buf
	if len(buf) < 5 {
		return 0, ErrMsgTooLarge
	}
	buf[4] = GetMessageType(m)
	buf = buf[5:]
	v := reflect.Indirect(reflect.ValueOf(m))
	for i := 0; i < v.NumField(); i++ {
		v := v.Field(i)
		switch v.Kind() {
		case reflect.Uint16:
			if len(buf) < 2 {
				return 0, ErrMsgTooLarge
			}
			binary.BigEndian.PutUint16(buf[0:2], uint16(v.Uint()))
			buf = buf[2:]
		case reflect.Uint32:
			if len(buf) < 4 {
				return 0, ErrMsgTooLarge
			}
			binary.BigEndian.PutUint32(buf[0:4], uint32(v.Uint()))
			buf = buf[4:]
		case reflect.Uint64:
			if len(buf) < 8 {
				return 0, ErrMsgTooLarge
			}
			binary.BigEndian.PutUint64(buf[0:8], uint64(v.Uint()))
			buf = buf[8:]
		case reflect.String:
			if len(buf) < 2 {
				return 0, ErrMsgTooLarge
			}
			str := v.String()
			sz := len(str)
			if sz > 65535 {
				return 0, ErrStrTooLarge
			}
			binary.BigEndian.PutUint16(buf[0:2], uint16(sz))
			buf = buf[2:]
			if len(buf) < sz {
				return 0, ErrMsgTooLarge
			}
			copy(buf, []byte(str))
			buf = buf[sz:]
		case reflect.Slice:
			if len(buf) < 2 {
				return 0, ErrMsgTooLarge
			}
			data := v.Bytes()
			sz := uint32(len(data))
			binary.BigEndian.PutUint32(buf[0:4], sz)
			buf = buf[4:]
			if uint32(len(buf)) < sz {
				return 0, ErrMsgTooLarge
			}
			copy(buf, data)
			buf = buf[sz:]
		default:
			panic("internal error")
		}
	}
	sz := len(origbuf) - len(buf)
	binary.BigEndian.PutUint32(origbuf[0:4], uint32(sz))
	return sz, nil
}
