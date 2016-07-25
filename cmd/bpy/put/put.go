package put

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/fs"
	"acha.ninja/bpy/fs/fsutil"
	"acha.ninja/bpy/remote"
	"encoding/hex"
	"flag"
	"log"
	"path"
	"path/filepath"
	"strings"
)

func Put() {
	tagArg := flag.String("tag", "default", "tag put data into")
	destArg := flag.String("dest", "/", "destination path")
	flag.Parse()

	if len(flag.Args()) != 1 {
		common.Die("please specify the local folder to put into dest\n")
	}
	srcPath, err := filepath.Abs(flag.Args()[0])
	if err != nil {
		common.Die("error getting src path: %s\n", err.Error())
	}
	destPath := *destArg

	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	log.Printf("getting remote...")
	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()
	wstore, err := common.GetCStoreWriter(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	rstore, err := common.GetCStoreReader(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	log.Printf("getting tag...")
	tagHash, ok, err := remote.GetTag(c, *tagArg)
	if err != nil {
		common.Die("error fetching tag hash: %s\n", err.Error())
	}
	var destHash [32]byte
	if ok {
		destHash, err = bpy.ParseHash(tagHash)
		if err != nil {
			common.Die("error parsing hash: %s\n", err.Error())
		}
	} else {
		empty, err := fs.EmptyDir(wstore, 0755)
		if err != nil {
			common.Die("error creating empty dir: %s\n", err.Error())
		}
		destHash = empty.Data
		err = wstore.Close()
		if err != nil {
			common.Die("error closing remote: %s\n", err.Error())
		}

		err = rstore.Close()
		if err != nil {
			common.Die("error closing remote: %s\n", err.Error())
		}
		err = remote.Tag(c, *tagArg, hex.EncodeToString(destHash[:]))
		if err != nil {
			common.Die("error creating tag: %s\n", err.Error())
		}
		common.Die("initialized tag rerun command")
	}

	log.Printf("copying host dir cstore...")
	srcDirEnt, err := fsutil.CpHostDirToFs(wstore, srcPath)
	if err != nil {
		common.Die("error copying data: %s\n", err.Error())
	}

	if strings.HasSuffix(destPath, "/") {
		srcDirEnt.EntName = filepath.Base(srcPath)
	} else {
		srcDirEnt.EntName = path.Base(destPath)
	}

	log.Printf("running insert... %s %s", srcDirEnt.EntName, destPath)
	newRoot, err := fs.Insert(rstore, wstore, destHash, destPath, srcDirEnt)
	if err != nil {
		common.Die("error inserting src into folder: %s\n", err.Error())
	}

	for {
		log.Printf("running cas...")
		ok, err := remote.CasTag(c, *tagArg, hex.EncodeToString(destHash[:]), hex.EncodeToString(newRoot.Data[:]))
		if err != nil {
			common.Die("creating tag: %s\n", err.Error())
		}
		if ok {
			log.Printf("cas done...")
			break
		}
	}

	log.Printf("shutting down...")
	err = wstore.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}
	log.Printf("shutting down...")
	err = rstore.Close()
	if err != nil {
		common.Die("error closing remote: %s\n", err.Error())
	}

	log.Printf("done...")
}
