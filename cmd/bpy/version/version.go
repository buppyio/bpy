package version

import (
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"runtime"
)

var BpyVersion string
var BpyCommit string

func Version() {
	_, err := fmt.Printf("bpy version: %s\n", BpyVersion)
	if err != nil {
		common.Die("error printing bpy version: %s\n", err)
	}
	fmt.Printf("bpy commit: %s\n", BpyCommit)
	if err != nil {
		common.Die("error printing bpy commit: %s\n", err)
	}
	fmt.Printf("go version: %s\n", runtime.Version())
	if err != nil {
		common.Die("error printing runtime version: %s\n", err)
	}
}
