package proto

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
)

const (
	RERROR = iota
	TATTACH
	RATTACH
	TSTAT
	RSTAT
	TOPEN
	ROPEN
	TREADAT
	RREADAT
	TCLOSE
	RCLOSE
	TNEWPACK
	RNEWPACK
	TWRITEPACK
	RPACKERROR
	TCLOSEPACK
	RCLOSEPACK
	TCANCELPACK
	RCANCELPACK
	TREMOVE
	RREMOVE
	TTAG
	RTAG
	TGETTAG
	RGETTAG
	TCASTAG
	RCASTAG
	TREMOVETAG
	RREMOVETAG
)

const (
	NOMID = 0
)

const (
	READOVERHEAD  = 4 + 1 + 2 + 4
	WRITEOVERHEAD = 4 + 1 + 4 + 4
)

var (
	ErrMsgTooLarge = errors.New("message too large")
	ErrStrTooLarge = errors.New("string too large")
	ErrMsgCorrupt  = errors.New("message corrupt")
)

type Message interface {
}

type RError struct {
	Mid     uint16
	Message string
}

type TTag struct {
	Mid   uint16
	Name  string
	Value string
}

type RTag struct {
	Mid uint16
}

type TGetTag struct {
	Mid  uint16
	Name string
}

type RGetTag struct {
	Mid   uint16
	Ok    bool
	Value string
}

type TCasTag struct {
	Mid      uint16
	Name     string
	OldValue string
	NewValue string
}

type RCasTag struct {
	Mid uint16
	Ok  bool
}

type TRemoveTag struct {
	Mid      uint16
	Name     string
	OldValue string
}

type RRemoveTag struct {
	Mid uint16
}

type TAttach struct {
	Mid            uint16
	MaxMessageSize uint32
	Version        string
	KeyId          string
}

type RAttach struct {
	Mid            uint16
	MaxMessageSize uint32
}

type TOpen struct {
	Mid  uint16
	Fid  uint32
	Name string
}

type ROpen struct {
	Mid uint16
}

type TReadAt struct {
	Mid    uint16
	Fid    uint32
	Offset uint64
	Size   uint32
}

type RReadAt struct {
	Mid  uint16
	Data []byte
}

type TClose struct {
	Mid uint16
	Fid uint32
}

type RClose struct {
	Mid uint16
}

type TNewPack struct {
	Mid  uint16
	Pid  uint32
	Name string
}

type RNewPack struct {
	Mid uint16
}

type TWritePack struct {
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

type TCancelPack struct {
	Mid uint16
	Pid uint32
}

type RCancelPack struct {
	Mid uint16
}

type TRemove struct {
	Mid  uint16
	Path string
}

type RRemove struct {
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

func WriteMessage(w io.Writer, m Message, buf []byte) error {
	n, err := PackMessage(m, buf)
	if err != nil {
		return nil
	}
	_, err = w.Write(buf[:n])
	return err
}

func UnpackMessage(buf []byte) (Message, error) {
	var m Message

	switch buf[4] {
	case RERROR:
		m = &RError{}
	case TATTACH:
		m = &TAttach{}
	case RATTACH:
		m = &RAttach{}
	case TOPEN:
		m = &TOpen{}
	case ROPEN:
		m = &ROpen{}
	case TREADAT:
		m = &TReadAt{}
	case RREADAT:
		m = &RReadAt{}
	case TCLOSE:
		m = &TClose{}
	case RCLOSE:
		m = &RClose{}
	case TNEWPACK:
		m = &TNewPack{}
	case RNEWPACK:
		m = &RNewPack{}
	case TWRITEPACK:
		m = &TWritePack{}
	case RPACKERROR:
		m = &RPackError{}
	case TCLOSEPACK:
		m = &TClosePack{}
	case RCLOSEPACK:
		m = &RClosePack{}
	case TCANCELPACK:
		m = &TCancelPack{}
	case RCANCELPACK:
		m = &RCancelPack{}
	case TTAG:
		m = &TTag{}
	case RTAG:
		m = &RTag{}
	case TGETTAG:
		m = &TGetTag{}
	case RGETTAG:
		m = &RGetTag{}
	case TCASTAG:
		m = &TCasTag{}
	case RCASTAG:
		m = &RCasTag{}
	case TREMOVETAG:
		m = &TRemoveTag{}
	case RREMOVETAG:
		m = &RRemoveTag{}
	default:
		return nil, ErrMsgCorrupt
	}
	return m, unpackFields(m, buf[5:])
}

func GetMessageType(m Message) byte {
	switch m.(type) {
	case *RError:
		return RERROR
	case *TAttach:
		return TATTACH
	case *RAttach:
		return RATTACH
	case *TOpen:
		return TOPEN
	case *ROpen:
		return ROPEN
	case *TReadAt:
		return TREADAT
	case *RReadAt:
		return RREADAT
	case *TClose:
		return TCLOSE
	case *RClose:
		return RCLOSE
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
	case *TCancelPack:
		return TCANCELPACK
	case *RCancelPack:
		return RCANCELPACK
	case *TTag:
		return TTAG
	case *RTag:
		return RTAG
	case *TGetTag:
		return TGETTAG
	case *RGetTag:
		return RGETTAG
	case *TRemoveTag:
		return TREMOVETAG
	case *RRemoveTag:
		return RREMOVETAG
	}
	panic("GetMessageType: internal error")
}

func GetMessageId(m Message) uint16 {
	switch m := m.(type) {
	case *RError:
		return m.Mid
	case *TAttach:
		return m.Mid
	case *RAttach:
		return m.Mid
	case *TOpen:
		return m.Mid
	case *ROpen:
		return m.Mid
	case *TReadAt:
		return m.Mid
	case *RReadAt:
		return m.Mid
	case *TClose:
		return m.Mid
	case *RClose:
		return m.Mid
	case *TNewPack:
		return m.Mid
	case *RNewPack:
		return m.Mid
	case *TWritePack:
		return NOMID
	case *RPackError:
		return NOMID
	case *TClosePack:
		return m.Mid
	case *RClosePack:
		return m.Mid
	case *TCancelPack:
		return m.Mid
	case *RCancelPack:
		return m.Mid
	case *TTag:
		return m.Mid
	case *RTag:
		return m.Mid
	case *TGetTag:
		return m.Mid
	case *RGetTag:
		return m.Mid
	case *TRemoveTag:
		return m.Mid
	case *RRemoveTag:
		return m.Mid
	}
	panic("GetMessageId: internal error")
}

func unpackFields(m Message, buf []byte) error {
	v := reflect.Indirect(reflect.ValueOf(m))
	for i := 0; i < v.NumField(); i++ {
		v := v.Field(i)
		switch v.Kind() {
		case reflect.Bool:
			if len(buf) < 1 {
				return ErrMsgCorrupt
			}
			v.SetBool(buf[0] != 0)
			buf = buf[1:]
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
			panic("unpackFields: internal error")
		}
	}
	if len(buf) != 0 {
		return ErrMsgCorrupt
	}
	return nil
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
		case reflect.Bool:
			if len(buf) < 1 {
				return 0, ErrMsgTooLarge
			}
			if v.Bool() {
				buf[0] = 1
			} else {
				buf[0] = 0
			}
			buf = buf[1:]
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
