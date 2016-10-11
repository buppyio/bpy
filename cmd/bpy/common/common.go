package common

import (
	"encoding/hex"
	"fmt"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cstore"
	"github.com/buppyio/bpy/cstore/cache"
	"github.com/buppyio/bpy/fs"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/remote/client"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

const (
	IdxCachePermissions = 0755
)

type slave struct {
	in  io.ReadCloser
	out io.WriteCloser
	cmd *exec.Cmd
}

func (s *slave) Read(buf []byte) (int, error) {
	return s.in.Read(buf)
}

func (s *slave) Write(buf []byte) (int, error) {
	return s.out.Write(buf)
}

func (s *slave) Close() error {
	s.in.Close()
	s.out.Close()
	return s.cmd.Process.Kill()
}

func dialRemote(cmdstr string) (io.ReadWriteCloser, error) {
	// XXX strings.Fields doesn't handle quotes correctly
	spltcmd := strings.Fields(cmdstr)
	if len(spltcmd) < 2 {
		return nil, fmt.Errorf("invalid remote command: '%s'", cmdstr)
	}
	cmd := exec.Command(spltcmd[0], spltcmd[1:]...)
	out, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	in, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	go io.Copy(os.Stderr, stderr)
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	return &slave{
		in:  in,
		out: out,
		cmd: cmd,
	}, nil
}

func GetBuppyDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, ".bpy"), nil
}

func GetIndexCacheDir() (string, error) {
	d, err := GetBuppyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "cache"), nil
}

func GetCacheDaemonAddress() (string, error) {
	return os.Getenv("BPY_CACHE"), nil
}

func GetCacheClient() (*cache.Client, bool, error) {
	addr, err := GetCacheDaemonAddress()
	if err != nil {
		return nil, false, err
	}

	if addr == "" {
		return nil, false, nil
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, false, err
	}

	cacheClient, err := cache.NewClient(conn)
	if err != nil {
		return nil, false, err
	}

	return cacheClient, true, nil
}

func GetKey() (bpy.Key, error) {
	d, err := GetBuppyDir()
	if err != nil {
		return bpy.Key{}, err
	}
	keyfile := filepath.Join(d, "bpy.key")
	f, err := os.Open(keyfile)
	if err != nil {
		return bpy.Key{}, err
	}
	defer f.Close()
	return bpy.ReadKey(f)
}

func GetRemote(k *bpy.Key) (*client.Client, error) {
	cmd := os.Getenv("BPY_REMOTE_CMD")
	slv, err := dialRemote(cmd)
	if err != nil {
		return nil, err
	}
	c, err := client.Attach(slv, hex.EncodeToString(k.Id[:]))
	if err != nil {
		return nil, err
	}
	_, ok, err := remote.GetRef(c, k)
	if err != nil {
		return nil, fmt.Errorf("error fetching ref: %s", err.Error())
	}
	if !ok {
		generation, err := remote.GetGeneration(c)
		if err != nil {
			return nil, err
		}

		store, err := GetCStore(k, c)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("error getting store writer: %s", err.Error())
		}

		ent, err := fs.EmptyDir(store, 0755)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("error creating empty default root: %s", err.Error())
		}

		ref := refs.Ref{
			CreatedAt: time.Now().Unix(),
			Root:      ent.HTree.Data,
		}

		hash, err := refs.PutRef(store, ref)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("error creating base ref: %s", err.Error())
		}

		err = store.Close()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("error closing store writer: %s", err.Error())
		}

		_, err = remote.CasRef(c, k, [32]byte{}, hash, generation)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("error initizializing root: %s", err.Error())
		}
	}
	return c, nil
}

func GetCStore(k *bpy.Key, remote *client.Client) (bpy.CStore, error) {
	var store bpy.CStore

	idxCache, err := GetIndexCacheDir()
	if err != nil {
		return nil, err
	}
	curIdxCache := filepath.Join(idxCache, hex.EncodeToString(k.Id[:]))
	err = os.MkdirAll(curIdxCache, IdxCachePermissions)
	if err != nil {
		return nil, err
	}
	store, err = cstore.NewWriter(remote, k.CipherKey, curIdxCache)
	if err != nil {
		return nil, err
	}

	cacheClient, ok, err := GetCacheClient()
	if err != nil {
		return nil, err
	}

	if ok {
		store = cstore.NewCachedCStore(store, cacheClient)
	}
	return store, nil
}

func Die(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}
