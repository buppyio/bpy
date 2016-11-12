package hist

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/buppyio/bpy"
	"github.com/buppyio/bpy/cmd/bpy/common"
	"github.com/buppyio/bpy/refs"
	"github.com/buppyio/bpy/remote"
	"github.com/buppyio/bpy/when"
	"os"
	"time"
)

func histHelp() {
	fmt.Println("Please specify one of the following subcommands:")
	fmt.Println("list, prune")
	os.Exit(1)
}

func prune() {
	pruneAllArg := flag.Bool("all", false, "prune all")
	pruneOlderThanArg := flag.String("older-than", "", "prune older than this time spec")

	flag.Parse()

	cfg, err := common.GetConfig()
	if err != nil {
		common.Die("error getting config: %s\n", err)
	}

	k, err := common.GetKey(cfg)
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(cfg, &k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	epoch, err := remote.GetEpoch(c)
	if err != nil {
		common.Die("error getting current epoch: %s\n", err.Error())
	}

	store, err := common.GetCStore(cfg, &k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	rootHash, rootVersion, ok, err := remote.GetRoot(c, &k)
	if err != nil {
		common.Die("error fetching root hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("root missing\n")
	}

	ref, err := refs.GetRef(store, rootHash)
	if err != nil {
		common.Die("error fetching ref: %s\n", err.Error())
	}

	if *pruneAllArg {
		newRef := ref
		newRef.HasPrev = false

		newRefHash, err := refs.PutRef(store, newRef)
		ok, err = remote.CasRoot(c, &k, newRefHash, bpy.NextRootVersion(rootVersion), epoch)
		if err != nil {
			common.Die("error swapping root: %s\n", err.Error())
		}
		if !ok {
			common.Die("root concurrently modified, try again\n")
		}
		err = store.Close()
		if err != nil {
			common.Die("error closing content store: %s\n", err.Error())
		}
		return
	}

	if *pruneOlderThanArg == "" {
		common.Die("please specify how much history to prune with -older-than")
	}

	pruneOlderThan, err := when.Parse(*pruneOlderThanArg)
	if err != nil {
		common.Die("error parsing prune time spec: %s\n", err.Error())
	}

	prunedHist := make([]refs.Ref, 0, 0)

	prunedHist = append(prunedHist, ref)
	for {
		ref, err = refs.GetRef(store, ref.Prev)
		if err != nil {
			common.Die("error fetching ref: %s\n", err.Error())
		}

		if !ref.HasPrev {
			break
		}

		if time.Unix(ref.CreatedAt, 0).Before(pruneOlderThan) {
			break
		}

		prunedHist = append(prunedHist, ref)
	}

	prunedHist[len(prunedHist)-1].HasPrev = false

	for i := len(prunedHist) - 1; i != 0; i-- {
		newRefHash, err := refs.PutRef(store, prunedHist[i])
		if err != nil {
			common.Die("error storing ref: %s\n", err.Error())
		}
		prunedHist[i-1].Prev = newRefHash
	}

	newRefHash, err := refs.PutRef(store, prunedHist[0])
	if err != nil {
		common.Die("error storing ref: %s\n", err.Error())
	}

	ok, err = remote.CasRoot(c, &k, newRefHash, bpy.NextRootVersion(rootVersion), epoch)
	if err != nil {
		common.Die("error swapping root: %s\n", err.Error())
	}
	if !ok {
		common.Die("ref concurrently modified, try again\n")
	}

	err = store.Close()
	if err != nil {
		common.Die("error closing content store: %s\n", err.Error())
	}
}

func list() {
	flag.Parse()

	cfg, err := common.GetConfig()
	if err != nil {
		common.Die("error getting config: %s\n", err)
	}

	k, err := common.GetKey(cfg)
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(cfg, &k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	store, err := common.GetCStore(cfg, &k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	rootHash, _, ok, err := remote.GetRoot(c, &k)
	if err != nil {
		common.Die("error fetching root hash: %s\n", err.Error())
	}
	if !ok {
		common.Die("root missing\n")
	}

	for {
		ref, err := refs.GetRef(store, rootHash)
		if err != nil {
			common.Die("error fetching ref: %s\n", err.Error())
		}
		_, err = fmt.Printf("%s@%s\n", hex.EncodeToString(rootHash[:]), time.Unix(ref.CreatedAt, 0))
		if err != nil {
			common.Die("io error: %s\n", err.Error())
		}
		if !ref.HasPrev {
			break
		}
		rootHash = ref.Prev
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
