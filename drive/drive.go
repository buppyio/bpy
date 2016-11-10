package drive

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"strconv"
	"time"
)

type PackListing struct {
	Name string
	Size uint64
	Date time.Time
}

type packState struct {
	UploadComplete bool
	GCGeneration   uint64
	Listing        PackListing
}

const (
	MetaDataBucketName = "metadata"
	PacksBucketName    = "packs"
)

var (
	ErrGCOccurred      = errors.New("concurrent garbage collection, operation failed")
	ErrGCNotRunning    = errors.New("garbage collection not running")
	ErrDuplicatePack   = errors.New("duplicate pack")
	ErrInvalidPackName = errors.New("invalid pack name")
)

type Drive struct {
	dbPath string
}

func openBoltDB(dbPath string) (*bolt.DB, error) {
	return bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
}

func nextGCGeneration(metaDataBucket *bolt.Bucket) (uint64, error) {
	gcGeneration, err := strconv.ParseUint(string(metaDataBucket.Get([]byte("gcgeneration"))), 10, 64)
	if err != nil {
		return 0, err
	}
	gcGeneration += 1
	err = metaDataBucket.Put([]byte("gcgeneration"), []byte(fmt.Sprintf("%d", gcGeneration)))
	if err != nil {
		return 0, err
	}
	return gcGeneration, nil
}

func Open(dbPath string) (*Drive, error) {
	db, err := openBoltDB(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket, err := tx.CreateBucketIfNotExists([]byte(MetaDataBucketName))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(PacksBucketName))
		if err != nil {
			return err
		}

		if string(metaDataBucket.Get([]byte("gcgeneration"))) == "" {
			err = metaDataBucket.Put([]byte("gcgeneration"), []byte("0"))
			if err != nil {
				return err
			}
		}

		if string(metaDataBucket.Get([]byte("gcrunning"))) == "" {
			err = metaDataBucket.Put([]byte("gcrunning"), []byte("0"))
			if err != nil {
				return err
			}
		}

		if string(metaDataBucket.Get([]byte("rootversion"))) == "" {
			err = metaDataBucket.Put([]byte("rootversion"), []byte("0"))
			if err != nil {
				return err
			}
		}

		if string(metaDataBucket.Get([]byte("rootsignature"))) == "" {
			err = metaDataBucket.Put([]byte("rootsignature"), []byte(""))
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &Drive{
		dbPath: dbPath,
	}, nil
}

func (d *Drive) Close() error {
	return nil
}

func (d *Drive) Attach(keyId string) (bool, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var ok bool
	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		currentKeyId := metaDataBucket.Get([]byte("keyid"))
		if currentKeyId != nil {
			if string(currentKeyId) != keyId {
				return nil
			}
		} else {
			err = metaDataBucket.Put([]byte("keyid"), []byte(keyId))
			if err != nil {
				return err
			}
		}
		ok = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (d *Drive) GetGCGeneration() (uint64, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var gcGenerationString string

	err = db.View(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		gcGenerationString = string(metaDataBucket.Get([]byte("gcgeneration")))
		return nil
	})

	if err != nil {
		return 0, err
	}

	gcGeneration, err := strconv.ParseUint(gcGenerationString, 10, 64)
	if err != nil {
		return 0, err
	}

	return gcGeneration, nil
}

func (d *Drive) StartGC() (uint64, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var gcGeneration uint64

	err = db.Update(func(tx *bolt.Tx) error {
		packsBucket := tx.Bucket([]byte(PacksBucketName))
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))

		gcGeneration, err = nextGCGeneration(metaDataBucket)
		if err != nil {
			return err
		}

		err = metaDataBucket.Put([]byte("gcrunning"), []byte("1"))
		if err != nil {
			return err
		}

		toDelete := [][]byte{}

		err = packsBucket.ForEach(func(k, v []byte) error {
			var state packState
			err := json.Unmarshal(v, &state)
			if err != nil {
				return err
			}
			if !state.UploadComplete {
				toDelete = append(toDelete, k)
			}
			return nil
		})
		if err != nil {
			return err
		}

		for _, packName := range toDelete {
			err := packsBucket.Delete(packName)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	err = db.Close()
	if err != nil {
		return 0, err
	}

	return gcGeneration, nil
}

func (d *Drive) StopGC() error {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))

		_, err = nextGCGeneration(metaDataBucket)
		if err != nil {
			return err
		}

		err = metaDataBucket.Put([]byte("gcrunning"), []byte("0"))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	err = db.Close()
	if err != nil {
		return err
	}

	return nil
}

func (d *Drive) CasRoot(root string, version uint64, signature string, gcGeneration uint64) (bool, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var ok bool

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		rootVersion, err := strconv.ParseUint(string(metaDataBucket.Get([]byte("rootversion"))), 10, 64)
		if err != nil {
			return err
		}

		if rootVersion+1 != version {
			return nil
		}

		curGCGeneration, err := strconv.ParseUint(string(metaDataBucket.Get([]byte("gcgeneration"))), 10, 64)
		if err != nil {
			return err
		}

		if curGCGeneration != gcGeneration {
			return nil
		}

		err = metaDataBucket.Put([]byte("rootversion"), []byte(fmt.Sprintf("%d", version)))
		if err != nil {
			return err
		}
		err = metaDataBucket.Put([]byte("rootval"), []byte(root))
		if err != nil {
			return err
		}
		err = metaDataBucket.Put([]byte("rootsignature"), []byte(signature))
		if err != nil {
			return err
		}

		ok = true
		return nil
	})

	if err != nil {
		return false, err
	}

	err = db.Close()
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (d *Drive) GetRoot() (string, uint64, string, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return "", 0, "", err
	}
	defer db.Close()

	var root, signature string
	var rootVersion uint64

	err = db.View(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		root = string(metaDataBucket.Get([]byte("rootval")))
		signature = string(metaDataBucket.Get([]byte("rootsignature")))
		rootVersion, err = strconv.ParseUint(string(metaDataBucket.Get([]byte("rootversion"))), 10, 64)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", 0, "", err
	}

	return root, rootVersion, signature, nil
}

func (d *Drive) StartUpload(packName string) error {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if len(packName) > 1024 {
		return errors.New("invalid pack name")
	}

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		packsBucket := tx.Bucket([]byte(PacksBucketName))

		if packsBucket.Get([]byte(packName)) != nil {
			return ErrDuplicatePack
		}

		curGCGeneration, err := strconv.ParseUint(string(metaDataBucket.Get([]byte("gcgeneration"))), 10, 64)
		if err != nil {
			return err
		}

		stateBytes, err := json.Marshal(packState{
			UploadComplete: false,
			GCGeneration:   curGCGeneration,
		})
		if err != nil {
			return err
		}

		err = packsBucket.Put([]byte(packName), stateBytes)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = db.Close()
	if err != nil {
		return err
	}

	return nil
}

func (d *Drive) FinishUpload(packName string, createdAt time.Time, size uint64) error {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if len(packName) > 1024 {
		return ErrInvalidPackName
	}

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		packsBucket := tx.Bucket([]byte(PacksBucketName))

		oldStateBytes := packsBucket.Get([]byte(packName))
		if oldStateBytes == nil {
			return ErrGCOccurred
		}

		var state packState

		err := json.Unmarshal(oldStateBytes, &state)
		if err != nil {
			return err
		}

		curGCGeneration, err := strconv.ParseUint(string(metaDataBucket.Get([]byte("gcgeneration"))), 10, 64)
		if err != nil {
			return err
		}

		if curGCGeneration != state.GCGeneration {
			return ErrGCOccurred
		}

		state.UploadComplete = true
		state.Listing.Date = createdAt
		state.Listing.Size = size

		newStateBytes, err := json.Marshal(state)
		if err != nil {
			return err
		}

		err = packsBucket.Put([]byte(packName), newStateBytes)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	err = db.Close()
	if err != nil {
		return err
	}

	return nil
}

func (d *Drive) RemovePack(packName string, gcGeneration uint64) error {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))

		if string(metaDataBucket.Get([]byte("gcrunning"))) != "1" {
			return ErrGCNotRunning
		}

		curGCGeneration, err := strconv.ParseUint(string(metaDataBucket.Get([]byte("gcgeneration"))), 10, 64)
		if err != nil {
			return err
		}

		if gcGeneration != curGCGeneration {
			return ErrGCOccurred
		}

		packsBucket := tx.Bucket([]byte(PacksBucketName))
		err = packsBucket.Delete([]byte(packName))
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = db.Close()
	if err != nil {
		return err
	}

	return nil
}

func (d *Drive) GetPacks() ([]PackListing, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	listing := make([]PackListing, 0, 32)
	err = db.View(func(tx *bolt.Tx) error {
		packsBucket := tx.Bucket([]byte(PacksBucketName))
		err = packsBucket.ForEach(func(k, v []byte) error {
			var state packState
			err := json.Unmarshal(v, &state)
			if err != nil {
				return err
			}
			if state.UploadComplete {
				state.Listing.Name = string(k)
				listing = append(listing, state.Listing)
			}
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return listing, nil
}
