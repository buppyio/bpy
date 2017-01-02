package common

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
)

var (
	DefaultCacheSocketType string
	DefaultCacheListenAddr string
)

type Config struct {
	BuppyPath       string
	RemoteCommand   string
	ICachePath      string
	CacheFile       string
	CacheSize       int64
	CacheSocketType string
	CacheListenAddr string
	KeyPath         string
}

func GetConfig() (*Config, error) {
	cfg := &Config{}

	err := setEnvConfigValues(cfg)
	if err != nil {
		return nil, err
	}
	err = setDefaultConfigValues(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func setEnvConfigValues(cfg *Config) error {
	if cfg.BuppyPath == "" {
		cfg.BuppyPath = os.Getenv("BPY_PATH")
	}
	if cfg.RemoteCommand == "" {
		cfg.RemoteCommand = os.Getenv("BPY_REMOTE_CMD")
	}
	if cfg.ICachePath == "" {
		cfg.ICachePath = os.Getenv("BPY_ICACHE_PATH")
	}
	if cfg.CacheFile == "" {
		cfg.CacheFile = os.Getenv("BPY_CACHE_FILE")
	}
	if cfg.CacheSocketType == "" {
		cfg.CacheSocketType = os.Getenv("BPY_CACHE_SOCKET_TYPE")
	}
	if cfg.CacheListenAddr == "" {
		cfg.CacheListenAddr = os.Getenv("BPY_CACHE_LISTEN_ADDR")
	}
	if cfg.CacheSize == 0 {
		szStr := os.Getenv("BPY_CACHE_SIZE")
		if szStr != "" {
			v, err := strconv.ParseInt(szStr, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing BPY_CACHE_SIZE (%s): %s", szStr, err)
			}
			cfg.CacheSize = v
		}
	}
	if cfg.KeyPath == "" {
		cfg.CacheFile = os.Getenv("BPY_KEY_PATH")
	}
	return nil
}

func setDefaultConfigValues(cfg *Config) error {
	if cfg.BuppyPath == "" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		cfg.BuppyPath = filepath.Join(u.HomeDir, ".bpy")
	}
	if cfg.ICachePath == "" {
		cfg.ICachePath = filepath.Join(cfg.BuppyPath, "icache")
	}
	if cfg.CacheFile == "" {
		cfg.CacheFile = filepath.Join(cfg.BuppyPath, "chunks.db")
	}
	if cfg.CacheSize == 0 {
		cfg.CacheSize = 512 * 1024 * 1024
	}
	if cfg.KeyPath == "" {
		cfg.KeyPath = filepath.Join(cfg.BuppyPath, "bpy.key")
	}
	switch runtime.GOOS {
	case "windows":
		if cfg.CacheSocketType == "" {
			cfg.CacheSocketType = "tcp"
		}
		if cfg.CacheListenAddr == "" {
			cfg.CacheListenAddr = "127.0.0.1:8877"
		}
	default:
		if cfg.CacheSocketType == "" {
			cfg.CacheSocketType = "unix"
		}
		if cfg.CacheListenAddr == "" {
			cfg.CacheListenAddr = filepath.Join(cfg.BuppyPath, "cache.sock")
		}
	}
	if cfg.CacheListenAddr == "" {
		cfg.CacheListenAddr = DefaultCacheListenAddr
	}
	return nil
}
