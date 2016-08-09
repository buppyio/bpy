package client

import (
	"github.com/buppyio/bpy/remote/proto"
)

type Pack struct {
	c   *Client
	pid uint32
}

func (p *Pack) Write(buf []byte) (int, error) {
	err := p.c.checkPidError(p.pid)
	if err != nil {
		return 0, err
	}
	maxn := p.c.getMaxMessageSize() - proto.WRITEOVERHEAD
	nsent := 0
	for len(buf) != 0 {
		n := maxn
		if uint32(len(buf)) < n {
			n = uint32(len(buf))
		}
		err := p.c.TWritePack(p.pid, buf[:n])
		if err != nil {
			return nsent, err
		}
		nsent += int(n)
		buf = buf[n:]
	}
	return nsent, nil
}

func (p *Pack) Close() error {
	p.c.freePid(p.pid)
	_, err := p.c.TClosePack(p.pid)
	return err
}

func (p *Pack) Cancel() error {
	p.c.freePid(p.pid)
	_, err := p.c.TCancelPack(p.pid)
	return err
}
