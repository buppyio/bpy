package server9

import (
	"acha.ninja/bpy/proto9"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"sync"
)

var (
	ErrNoSuchFid     = errors.New("no such fid")
	ErrFidInUse      = errors.New("fid in use")
	ErrBadFid        = errors.New("bad fid")
	ErrBadTag        = errors.New("bad tag")
	ErrBadPath       = errors.New("bad path")
	ErrNotDir        = errors.New("not a directory path")
	ErrNotExist      = errors.New("no such file")
	ErrFileNotOpen   = errors.New("file not open")
	ErrBadReadOffset = errors.New("bad read offset")
)

type File interface {
	Parent() (File, error)
	Child(name string) (File, error)
	Qid() (proto9.Qid, error)
	Stat() (proto9.Stat, error)
}

func Walk(f File, names []string) (File, []proto9.Qid, error) {
	var werr error
	wqids := make([]proto9.Qid, 0, len(names))

	i := 0
	name := ""
	for i, name = range names {
		if name == "." || name == "" || strings.Index(name, "/") != -1 {
			return nil, nil, ErrBadPath
		}
		if name == ".." {
			parent, err := f.Parent()
			if err != nil {
				return nil, nil, err
			}
			qid, err := parent.Qid()
			if err != nil {
				return nil, nil, err
			}
			f = parent
			wqids = append(wqids, qid)
			continue
		}
		qid, err := f.Qid()
		if err != nil {
			return nil, nil, err
		}
		if !qid.IsDir() {
			werr = ErrNotDir
			goto walkerr
		}
		child, err := f.Child(name)
		if err != nil {
			if err == ErrNotExist {
				werr = ErrNotExist
				goto walkerr
			}
			return nil, nil, err
		}
		f = child
		wqids = append(wqids, qid)
	}
	return f, wqids, nil

walkerr:
	if i == 0 {
		return nil, nil, werr
	}
	return nil, wqids, nil
}

func ReadMsg(r io.Reader, buf []byte) (proto9.Msg, error) {
	if len(buf) < 5 {
		return nil, proto9.ErrBuffTooSmall
	}
	_, err := r.Read(buf[0:5])
	if err != nil {
		return nil, err
	}
	sz := int(binary.LittleEndian.Uint16(buf[0:4]))
	if len(buf) < sz {
		return nil, proto9.ErrBuffTooSmall
	}
	_, err = r.Read(buf[5:sz])
	if err != nil {
		return nil, err
	}
	return proto9.UnpackMsg(buf[0:sz])
}

func WriteMsg(w io.Writer, buf []byte, msg proto9.Msg) error {
	packed, err := proto9.PackMsg(buf, msg)
	if err != nil {
		return err
	}
	_, err = w.Write(packed)
	if err != nil {
		return err
	}
	return nil
}

func MakeError(t proto9.Tag, err error) proto9.Msg {
	return &proto9.Rerror{
		Tag: t,
		Err: err.Error(),
	}
}

var pathMutex sync.Mutex
var pathCount uint64

func NextPath() uint64 {
	pathMutex.Lock()
	r := pathCount
	pathCount++
	pathMutex.Unlock()
	return r
}

type StatList struct {
	Offset uint64
	Stats  []proto9.Stat
}

func (sl *StatList) ReadAt(buf []byte, off uint64) (int, error) {
	if off != sl.Offset {
		return 0, ErrBadReadOffset
	}
	n := 0
	for {
		if len(sl.Stats) == 0 {
			break
		}
		curstat := sl.Stats[0]
		statlen := proto9.StatLen(&curstat)
		if uint64(statlen+n) > uint64(len(buf)) {
			if n == 0 {
				return 0, proto9.ErrBuffTooSmall
			}
			break
		}
		proto9.PackStat(buf[n:n+statlen], &curstat)
		n += statlen
		sl.Stats = sl.Stats[1:]
	}
	sl.Offset += uint64(n)
	return n, nil
}
