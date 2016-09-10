package hist

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"os"
	"time"
)

func histHelp() {
	fmt.Println("Please specify one of the following subcommands:")
	fmt.Println("list, prune")
	os.Exit(1)
}

func prune() {
	refArg := flag.String("ref", "default", "ref to fetch history for")
	pruneAllArg := flag.Bool("all", false, "prune all")
	// pruneOlderThan := flag.String("older-than", "", "prune older than this time spec")

	flag.Parse()

	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	generation, err := remote.GetGeneration(c)
	if err != nil {
		common.Die("error getting current gc generation: %s\n", err.Error())
	}

	store, err := common.GetCStore(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	refHash, ok, err := remote.GetNamedRef(c, &k, *refArg)
	if err != nil {
		common.Die("error fetching ref hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("ref '%s' does not exist\n", *refArg)
	}

	ref, err := refs.GetRef(store, refHash)
	if err != nil {
		common.Die("error fetching ref: %s\n", err.Error())
	}

	if *pruneAllArg {
		newRef := ref
		newRef.HasPrev = false

		newRefHash, err := refs.PutRef(store, newRef)
		ok, err = remote.CasNamedRef(c, &k, *refArg, refHash, newRefHash, generation)
		if err != nil {
			common.Die("error swapping ref: %s\n", err.Error())
		}
		if !ok {
			common.Die("ref concurrently modified, try again\n")
		}
		err = store.Close()
		if err != nil {
			common.Die("error closing content store: %s\n", err.Error())
		}
		return
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing content store: %s\n", err.Error())
	}
}

func list() {
	refArg := flag.String("ref", "default", "ref to fetch history for")

	flag.Parse()

	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	store, err := common.GetCStore(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	refHash, ok, err := remote.GetNamedRef(c, &k, *refArg)
	if err != nil {
		common.Die("error fetching ref hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("ref '%s' does not exist\n", *refArg)
	}

	for {
		ref, err := refs.GetRef(store, refHash)
		if err != nil {
			common.Die("error fetching ref: %s\n", err.Error())
		}
		_, err = fmt.Printf("%s@%s\n", hex.EncodeToString(refHash[:]), time.Unix(ref.CreatedAt, 0))
		if err != nil {
			common.Die("io error: %s\n", err.Error())
		}
		if !ref.HasPrev {
			break
		}
		refHash = ref.Prev
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing content store: %s\n", err.Error())
	}
}

func Hist() {
	cmd := histHelp
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			cmd = list
		case "prune":
			cmd = prune
		default:
		}
		copy(os.Args[1:], os.Args[2:])
		os.Args = os.Args[0 : len(os.Args)-1]
	}
	cmd()
}
