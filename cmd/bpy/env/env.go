package env

import (
	"flag"
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/common"
)

func Env() {
	flag.Parse()

	cfg, err := common.GetConfig()
	if err != nil {
		common.Die("error getting config: %s\n", err)
	}

	errMsg := "error printing env: %s\n"

	_, err = fmt.Printf("BPY_REMOTE_CMD=%s\n", cfg.RemoteCommand)
	if err != nil {
		common.Die(errMsg, err)
	}
	_, err = fmt.Printf("BPY_PATH=%s\n", cfg.BuppyPath)
	if err != nil {
		common.Die(errMsg, err)
	}
	_, err = fmt.Printf("BPY_ICACHE_PATH=%s\n", cfg.ICachePath)
	if err != nil {
		common.Die(errMsg, err)
	}
	_, err = fmt.Printf("BPY_CACHE_FILE=%s\n", cfg.CacheFile)
	if err != nil {
		common.Die(errMsg, err)
	}
	_, err = fmt.Printf("BPY_CACHE_SIZE=%d\n", cfg.CacheSize)
	if err != nil {
		common.Die(errMsg, err)
	}
	_, err = fmt.Printf("BPY_CACHE_LISTEN_ADDR=%s\n", cfg.CacheListenAddr)
	if err != nil {
		common.Die(errMsg, err)
	}
}
