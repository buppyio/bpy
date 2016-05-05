package common

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/client9"
	"acha.ninja/bpy/cstore"
	"acha.ninja/bpy/proto9"
	"io"
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

func GetStore() (*client9.Client, error) {
	slv, err := dialRemote("acha.ninja", "/home/ac/bpy")
	if err != nil {
		return nil, err
	}
	store, err := client9.NewClient(proto9.NewConn(slv, slv))
	if err != nil {
		return nil, err
	}
	err = store.Attach("ac", "")
	if err != nil {
		return nil, err
	}
	return store, nil
}

func GetCStoreReader() (bpy.CStoreReader, error) {
	store, err := GetStore()
	if err != nil {
		return nil, err
	}
	cache, err := GetCacheDir()
	if err != nil {
		return nil, err
	}
	return cstore.NewReader(store, cache)
}

func GetCStoreWriter() (bpy.CStoreWriter, error) {
	store, err := GetStore()
	if err != nil {
		return nil, err
	}
	cache, err := GetCacheDir()
	if err != nil {
		return nil, err
	}
	return cstore.NewWriter(store, cache)
}
