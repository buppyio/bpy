package common

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/remote"
	"acha.ninja/bpy/remote/client"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
)

const (
	CachePermissions = 0755
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

func dialRemote(remote string) (io.ReadWriteCloser, error) {
	url, path, err := ParseRemoteString(remote)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("ssh", url, "bpy", "remote", path)
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

func GetCacheDir() (string, error) {
	d, err := GetBuppyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "cache"), nil
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
	url := os.Getenv("BPY_REMOTE")
	slv, err := dialRemote(url)
	if err != nil {
		return nil, err
	}
	c, err := client.Attach(slv, hex.EncodeToString(k.Id[:]))
	if err != nil {
		return nil, err
	}
	_, ok, err := remote.GetTag(c, "default")
	if !ok {
		k, err := GetKey()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("error getting key: %s", err.Error())
		}
		w, err := GetCStoreWriter(&k, c)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("error getting store writer: %s", err.Error())
		}
		ent, err := fs.EmptyDir(w, 0755)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("error creating empty default root: %s", err.Error())
		}
		err = w.Close()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("error closing writer: %s", err.Error())
		}
		err = remote.Tag(c, "default", hex.EncodeToString(ent.Data[:]))
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("error initizializing default tag: %s", err.Error())
		}
	}
	return c, nil
}

func GetCStoreReader(k *bpy.Key, remote *client.Client) (bpy.CStoreReader, error) {
	cache, err := GetCacheDir()
	if err != nil {
		return nil, err
	}
	curCache := filepath.Join(cache, hex.EncodeToString(k.Id[:]))
	err = os.MkdirAll(curCache, CachePermissions)
	if err != nil {
		return nil, err
	}
	return cstore.NewReader(remote, k.CipherKey, curCache)
}

func GetCStoreWriter(k *bpy.Key, remote *client.Client) (bpy.CStoreWriter, error) {
	cache, err := GetCacheDir()
	if err != nil {
		return nil, err
	}
	curCache := filepath.Join(cache, hex.EncodeToString(k.Id[:]))
	err = os.MkdirAll(curCache, CachePermissions)
	if err != nil {
		return nil, err
	}
	return cstore.NewWriter(remote, k.CipherKey, filepath.Join(cache, hex.EncodeToString(k.Id[:])))
}

func Die(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}

func ParseRemoteString(remote string) (string, string, error) {
	r := regexp.MustCompile("ssh://([^/]+)(.+)")

	matches := r.FindStringSubmatch(remote)
	if matches == nil {
		return "", "", fmt.Errorf("invalid remote: '%s'\n", remote)
	}
	return matches[1], matches[2], nil
}
