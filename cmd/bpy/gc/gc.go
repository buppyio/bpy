package gc

import (
	"acha.ninja/bpy"
	"acha.ninja/bpy/bpack"
	"acha.ninja/bpy/cmd/bpy/common"
	"acha.ninja/bpy/remote"
	"log"
	"path"
)

func GC() {

	k, err := common.GetKey()
	if err != nil {
		common.Die("error getting bpy key data: %s\n", err.Error())
	}

	c, err := common.GetRemote(&k)
	if err != nil {
		common.Die("error connecting to remote: %s\n", err.Error())
	}
	defer c.Close()

	_, err = common.GetCStore(&k, c)
	if err != nil {
		common.Die("error getting content store: %s\n", err.Error())
	}

	// Stop any gc that is currently running
	err = remote.StopGC(c)
	if err != nil {
		common.Die("error stopping gc: %s\n", err.Error())
	}

	gcId, err := remote.StartGC(c)
	if err != nil {
		common.Die("error starting gc: %s\n", err.Error())
	}

	packs, err := remote.ListPacks(c)
	if err != nil {
		remote.StopGC(c)
		common.Die("error listing packs: %s\n", err.Error())
	}
	var newPack *bpack.Writer
	newPackSize := uint64(0)
	canDelete := []string{}
	for _, pack := range packs {
		packPath := path.Join("packs/", pack.Name)
		log.Printf("packPath=%s", packPath)
		f, err := c.Open(packPath)
		if err != nil {
			common.Die("error opening pack file: %s\n", err.Error())
		}
		packReader, err := bpack.NewEncryptedReader(f, k.CipherKey, int64(pack.Size))
		if err != nil {
			common.Die("error opening pack file: %s\n", err.Error())
		}
		err = packReader.ReadIndex()
		if err != nil {
			common.Die("error reading pack file index %s\n", err.Error())
		}
		idx := packReader.Idx
		for _, idxEnt := range idx {
			if newPackSize+uint64(idxEnt.Size) > 128*1024*1024 {
				_, err := newPack.Close()
				if err != nil {
					common.Die("error closing new pack %s\n", err.Error())
				}
				newPack = nil
				newPackSize = 0
				for _, toDelete := range canDelete {
					err := remote.Remove(c, toDelete, gcId)
					if err != nil {
						common.Die("error removing old pack %s\n", err.Error())
					}
				}
				canDelete = []string{}
			}
			if newPack == nil {
				name, err := bpy.RandomFileName()
				if err != nil {
					common.Die("error generating new pack name %s\n", err.Error())
				}
				f, err := c.NewPack(path.Join("packs", name) + ".ebpack")
				if err != nil {
					common.Die("error making new pack %s\n", err.Error())
				}
				newPack, err = bpack.NewEncryptedWriter(f, k.CipherKey)
				if err != nil {
					common.Die("error making new pack %s\n", err.Error())
				}
			}
			val, err := packReader.Get(idxEnt.Key)
			if err != nil {
				common.Die("error reading pack value\n", err.Error())
			}
			err = newPack.Add(idxEnt.Key, val)
			if err != nil {
				common.Die("error adding value to new pack\n", err.Error())
			}
			// Only approximate, but good enough.
			newPackSize += uint64(len(idxEnt.Key)) + uint64(len(val))
		}
		err = packReader.Close()
		if err != nil {
			common.Die("error closing pack file %s\n", err.Error())
		}
		canDelete = append(canDelete, packPath)
	}
	_, err = newPack.Close()
	if err != nil {
		common.Die("error closing new pack %s\n", err.Error())
	}
	for _, toDelete := range canDelete {
		err := remote.Remove(c, toDelete, gcId)
		if err != nil {
			common.Die("error removing old pack %s\n", err.Error())
		}
	}

	err = remote.StopGC(c)
	if err != nil {
		common.Die("error stopping gc: %s\n", err.Error())
	}

}
