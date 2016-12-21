package p9

import (
	"errors"
	"fmt"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/p9/proto9"
	"github.com/buppyio/bpy/cmd/bpy/p9/server9"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/htree"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/remote/client"
	"io"
	"path"
	"time"
)

var (
	ErrReadOnly = errors.New("read only")
)

type fs9 struct {
	key         bpy.Key
	store       bpy.CStore
	client      *client.Client
	pathCounter uint64
	version     uint32
	file        *file
	lastUpdated time.Time
}

func (fs *fs9) CreateFile(ent fs.DirEnt, parent server9.File, fspath string) (*file, error) {

	fs.pathCounter++

	mode := proto9.FileMode(ent.Mode() & 0777)
	qtype := proto9.QTFILE

	if ent.Mode().IsDir() {
		mode |= proto9.DMDIR
		qtype = proto9.QTDIR
	}

	qid := proto9.Qid{
		Type:    qtype,
		Path:    fs.pathCounter,
		Version: fs.version,
	}

	f := &file{
		fs:     fs,
		parent: parent,
		path:   fspath,
		qid:    qid,
		stat: proto9.Stat{
			Mode:   mode,
			Atime:  0,
			Mtime:  0,
			Name:   ent.Name(),
			Qid:    qid,
			Length: uint64(ent.Size()),
			UID:    "nobody",
			GID:    "nobody",
			MUID:   "nobody",
		},
		ent:      ent,
		children: nil,
	}

	return f, nil
}

func (r *fs9) update() error {

	if r.file != nil && time.Since(r.lastUpdated) < 30*time.Second {
		return nil
	}

	root, _, ok, err := remote.GetRoot(r.client, &r.key)
	if err != nil {
		return fmt.Errorf("error getting root: %s", err)
	}
	if !ok {
		return fmt.Errorf("root missing\n")
	}

	ref, err := refs.GetRef(r.store, root)
	if err != nil {
		return fmt.Errorf("error getting ref: %s", err)
	}

	if r.file != nil && ref.Root == r.file.ent.HTree.Data {
		return nil
	}

	dirEnts, err := fs.ReadDir(r.store, ref.Root)
	if err != nil {
		return fmt.Errorf("error reading root: %s", err)
	}

	f, err := r.CreateFile(dirEnts[0], nil, "/")
	if err != nil {
		return fmt.Errorf("error creating root file: %s", err)
	}
	r.file = f

	r.lastUpdated = time.Now()
	r.version++

	return nil
}

func (r *fs9) Parent() (server9.File, error) {
	err := r.update()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (r *fs9) Child(name string) (server9.File, error) {
	err := r.update()
	if err != nil {
		return nil, err
	}

	return r.file.Child(name)
}

func (r *fs9) Qid() (proto9.Qid, error) {
	err := r.update()
	if err != nil {
		return proto9.Qid{}, err
	}

	return proto9.Qid{
		Type:    proto9.QTDIR,
		Path:    0,
		Version: r.version,
	}, nil
}

func (r *fs9) Stat() (proto9.Stat, error) {
	err := r.update()
	if err != nil {
		return proto9.Stat{}, err
	}

	qid, err := r.file.Qid()
	if err != nil {
		return proto9.Stat{}, err
	}
	st, err := r.file.Stat()
	if err != nil {
		return proto9.Stat{}, err
	}
	st.Qid = qid
	return st, nil
}

func (r *fs9) NewHandle() (server9.Handle, error) {
	err := r.update()
	if err != nil {
		return nil, err
	}

	return &rootHandle{
		fs: r,
	}, nil
}

type rootHandle struct {
	fs     *fs9
	offset uint64
	stats  []proto9.Stat
}

func (h *rootHandle) GetFile() (server9.File, error) {
	err := h.fs.update()
	if err != nil {
		return nil, err
	}

	return h.fs.file, nil
}

func (h *rootHandle) GetIounit(maxMessageSize uint32) uint32 {
	return maxMessageSize - proto9.WriteOverhead
}

func (h *rootHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	err := h.fs.update()
	if err != nil {
		return nil, nil, err
	}
	return server9.Walk(h.fs.file, msg.Names)
}

func (h *rootHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	err := h.fs.update()
	if err != nil {
		return proto9.Qid{}, err
	}

	return h.fs.file.qid, nil
}

func (h *rootHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	err := h.fs.update()
	if err != nil {
		return 0, err
	}

	if msg.Offset == 0 {
		children, err := h.fs.file.getChildren()
		if err != nil {
			return 0, err
		}
		h.stats = make([]proto9.Stat, len(children), len(children))
		for i, child := range children {
			h.stats[i] = child.stat
		}
	}

	if msg.Offset != h.offset {
		return 0, server9.ErrBadRead
	}

	n := uint32(0)
	for len(h.stats) != 0 {
		curstat := h.stats[0]
		statlen := uint32(proto9.StatLen(&curstat))
		if uint64(statlen+n) > uint64(len(buf)) {
			if n == 0 {
				return 0, proto9.ErrBuffTooSmall
			}
			break
		}
		proto9.PackStat(buf[n:n+statlen], &curstat)
		n += statlen
		h.stats = h.stats[1:]
	}
	h.offset += uint64(n)
	return n, nil
}

func (h *rootHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	return 0, ErrReadOnly
}

func (h *rootHandle) Tcreate(msg *proto9.Tcreate) (server9.Handle, error) {
	return nil, ErrReadOnly
}

func (h *rootHandle) Twstat(msg *proto9.Twstat) error {
	return ErrReadOnly
}

func (h *rootHandle) Tremove(msg *proto9.Tremove) error {
	return ErrReadOnly
}

func (h *rootHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	err := h.fs.update()
	if err != nil {
		return proto9.Stat{}, err
	}

	return h.fs.Stat()
}

func (h *rootHandle) Clunk() error {
	return nil
}

type file struct {
	fs       *fs9
	parent   server9.File
	path     string
	qid      proto9.Qid
	stat     proto9.Stat
	ent      fs.DirEnt
	children []*file
}

func (f *file) getChildren() ([]*file, error) {
	if !f.ent.IsDir() {
		return nil, server9.ErrNotDir
	}

	ents, err := fs.ReadDir(f.fs.store, f.ent.HTree.Data)
	if err != nil {
		return nil, err
	}
	ents = ents[1:]
	children := []*file{}

	for _, ent := range ents {
		child, err := f.fs.CreateFile(ent, f, path.Join(f.path, ent.EntName))
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}

	return children, nil
}

func (f *file) Parent() (server9.File, error) {
	return f.parent, nil
}

func (f *file) Child(name string) (server9.File, error) {
	if !f.ent.IsDir() {
		return nil, fmt.Errorf("%s is not a dir", f.path)
	}

	children, err := f.getChildren()
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(children); i++ {
		if children[i].ent.EntName == name {
			return children[i], nil
		}
	}

	return nil, fmt.Errorf("%s does not exist", path.Join(f.path, name))
}

func (f *file) Qid() (proto9.Qid, error) {
	return f.qid, nil
}

func (f *file) Stat() (proto9.Stat, error) {
	return f.stat, nil
}

func (f *file) NewHandle() (server9.Handle, error) {
	if f.ent.IsDir() {
		return &dirHandle{
			file: f,
		}, nil
	}
	return &fileHandle{
		file: f,
	}, nil
}

type dirHandle struct {
	file   *file
	offset uint64
	stats  []proto9.Stat
}

func (d *dirHandle) GetFile() (server9.File, error) {
	return d.file, nil
}

func (d *dirHandle) GetIounit(maxMessageSize uint32) uint32 {
	return maxMessageSize - proto9.WriteOverhead
}

func (d *dirHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	return server9.Walk(d.file, msg.Names)
}

func (d *dirHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	return d.file.qid, nil
}

func (d *dirHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	if msg.Offset == 0 {
		children, err := d.file.getChildren()
		if err != nil {
			return 0, err
		}
		d.stats = make([]proto9.Stat, len(children), len(children))
		for i, child := range children {
			d.stats[i] = child.stat
		}
	}

	if msg.Offset != d.offset {
		return 0, server9.ErrBadRead
	}
	n := uint32(0)
	for len(d.stats) != 0 {
		curstat := d.stats[0]
		statlen := uint32(proto9.StatLen(&curstat))
		if uint64(statlen+n) > uint64(len(buf)) {
			if n == 0 {
				return 0, proto9.ErrBuffTooSmall
			}
			break
		}
		proto9.PackStat(buf[n:n+statlen], &curstat)
		n += statlen
		d.stats = d.stats[1:]
	}
	d.offset += uint64(n)
	return n, nil
}

func (d *dirHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	return 0, ErrReadOnly
}

func (d *dirHandle) Tcreate(msg *proto9.Tcreate) (server9.Handle, error) {
	return nil, ErrReadOnly
}

func (d *dirHandle) Twstat(msg *proto9.Twstat) error {
	return ErrReadOnly
}

func (d *dirHandle) Tremove(msg *proto9.Tremove) error {
	return ErrReadOnly
}

func (d *dirHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return d.file.Stat()
}

func (d *dirHandle) Clunk() error {
	return nil
}

type fileHandle struct {
	file *file
	rdr  *htree.Reader
}

func (f *fileHandle) GetFile() (server9.File, error) {
	return f.file, nil
}

func (f *fileHandle) GetIounit(maxMessageSize uint32) uint32 {
	return maxMessageSize - proto9.WriteOverhead
}

func (f *fileHandle) Twalk(msg *proto9.Twalk) (server9.File, []proto9.Qid, error) {
	return server9.Walk(f.file, msg.Names)
}

func (f *fileHandle) Topen(msg *proto9.Topen) (proto9.Qid, error) {
	if f.rdr != nil {
		f.rdr = nil
	}
	var err error
	f.rdr, err = htree.NewReader(f.file.fs.store, f.file.ent.HTree.Data)
	return f.file.qid, err
}

func (f *fileHandle) Tread(msg *proto9.Tread, buf []byte) (uint32, error) {
	if f.rdr == nil {
		return 0, fmt.Errorf("fid for '%s' is not open", f.file.path)
	}
	n, err := f.rdr.Read(buf)
	if n != 0 {
		return uint32(n), nil
	}
	if err == io.EOF {
		return 0, nil
	}
	return 0, err
}

func (f *fileHandle) Twrite(msg *proto9.Twrite) (uint32, error) {
	return 0, ErrReadOnly
}

func (f *fileHandle) Tcreate(msg *proto9.Tcreate) (server9.Handle, error) {
	return nil, ErrReadOnly
}

func (f *fileHandle) Twstat(msg *proto9.Twstat) error {
	return ErrReadOnly
}

func (f *fileHandle) Tremove(msg *proto9.Tremove) error {
	return ErrReadOnly
}

func (f *fileHandle) Tstat(msg *proto9.Tstat) (proto9.Stat, error) {
	return f.file.Stat()
}

func (f *fileHandle) Clunk() error {
	if f.rdr != nil {
		f.rdr = nil
	}
	return nil
}
