package common

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/proto9"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
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

func dialRemote(url, path string) (io.ReadWriteCloser, error) {
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

func GetRemote(k *bpy.Key) (*client9.Client, error) {
	slv, err := dialRemote("localhost", "/home/ac/bpy")
	if err != nil {
		return nil, err
	}
	remote, err := client9.NewClient(proto9.NewConn(slv, slv))
	if err != nil {
		return nil, err
	}
	err = remote.Attach("nobody", hex.EncodeToString(k.Id[:]))
	if err != nil {
		return nil, err
	}
	return remote, nil
}

func GetCStoreReader(k *bpy.Key, remote *client9.Client) (bpy.CStoreReader, error) {
	cache, err := GetCacheDir()
	if err != nil {
		return nil, err
	}
	return cstore.NewReader(remote, k.CipherKey, filepath.Join(cache, hex.EncodeToString(k.Id[:])))
}

func GetCStoreWriter(k *bpy.Key, remote *client9.Client) (bpy.CStoreWriter, error) {
	cache, err := GetCacheDir()
	if err != nil {
		return nil, err
	}
	return cstore.NewWriter(remote, k.CipherKey, filepath.Join(cache, hex.EncodeToString(k.Id[:])))
}

func Die(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}
