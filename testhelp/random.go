package testhelp

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
)

type RandDirConfig struct {
	MaxDepth    uint32
	MaxFiles    uint32
	MaxSubdirs  uint32
	MaxFileSize uint32
}

var pathchars = []string{
	"a", "b", "c", "d", "e", "f", "h", " ", "\t", "ф",
}

func randomPathChar() string {
	return pathchars[int(rand.Uint32()%uint32(len(pathchars)))]
}

func randomName() string {
	l := int(rand.Uint32()%10) + 1
	name := ""
	for i := 0; i < l; i++ {
		name += randomPathChar()
	}
	return name
}

func RandomDirectoryTree(p string, cfg RandDirConfig, rd *rand.Rand) error {
	if cfg.MaxFileSize > 0 {
		nfiles := rand.Uint32() % cfg.MaxFiles
		for i := uint32(0); i < nfiles; i++ {
			fsz := rand.Uint32() % cfg.MaxFileSize
			name := fmt.Sprintf("file%d", i) + randomName()
			perms := []os.FileMode{0777, 0666}
			perm := perms[(int(rand.Int63()) % len(perms))]
			f, err := os.OpenFile(path.Join(p, name), os.O_CREATE|os.O_WRONLY, perm)
			if err != nil {
				return err
			}
			_, err = io.CopyN(f, rd, int64(fsz))
			if err != nil {
				return err
			}
			err = f.Close()
			if err != nil {
				return err
			}
		}
	}
	if cfg.MaxSubdirs > 0 && cfg.MaxDepth > 0 {
		ndirs := rand.Uint32() % cfg.MaxSubdirs
		for i := uint32(0); i < ndirs; i++ {
			newcfg := cfg
			newcfg.MaxDepth -= 1
			newp := path.Join(p, fmt.Sprintf("subdir%d", i)+randomName())
			os.Mkdir(newp, 0700)
			err := RandomDirectoryTree(newp, newcfg, rd)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
