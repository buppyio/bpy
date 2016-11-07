package drive

import (
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"strconv"
	"time"
)

const (
	MetaDataBucketName = "metadata"
	PacksBucketName    = "packs"
)

type Drive struct {
	dbPath string
}

func openBoltDB(dbPath string) (*bolt.DB, error) {
	return bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
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

		if string(metaDataBucket.Get([]byte("rootversion"))) == "" {
			err = metaDataBucket.Put([]byte("rootversion"), []byte("0"))
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

func (d *Drive) GetGCGeneration() (int64, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return -1, err
	}
	defer db.Close()

	var gcGenerationString string

	err = db.View(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		gcGenerationString = string(metaDataBucket.Get([]byte("gcgeneration")))
		return nil
	})

	if err != nil {
		return -1, err
	}

	gcGeneration, err := strconv.ParseInt(gcGenerationString, 10, 64)
	if err != nil {
		return -1, err
	}

	return gcGeneration, nil
}

func (d *Drive) StartGC() (int64, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return -1, err
	}
	defer db.Close()

	var gcGeneration int64

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		gcGeneration, err = strconv.ParseInt(string(metaDataBucket.Get([]byte("gcgeneration"))), 10, 64)
		if err != nil {
			return err
		}

		gcGeneration += 1
		err = metaDataBucket.Put([]byte("gcgeneration"), []byte(fmt.Sprintf("%d", gcGeneration)))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return -1, err
	}

	return gcGeneration, nil
}

func (d *Drive) StopGC() error {
	_, err := d.StartGC()
	return err
}

func (d *Drive) CasRoot(root string, newVersion, gcGeneration int64) (bool, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var ok bool

	err = db.Update(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		rootVersion, err := strconv.ParseInt(string(metaDataBucket.Get([]byte("rootversion"))), 10, 64)
		if err != nil {
			return err
		}

		rootVersion += 1
		if rootVersion != newVersion {
			return nil
		}

		curGCGeneration, err := strconv.ParseInt(string(metaDataBucket.Get([]byte("rootversion"))), 10, 64)
		if err != nil {
			return err
		}

		if curGCGeneration != gcGeneration {
			return nil
		}

		err = metaDataBucket.Put([]byte("rootversion"), []byte(fmt.Sprintf("%d", newVersion)))
		if err != nil {
			return err
		}
		err = metaDataBucket.Put([]byte("rootval"), []byte(root))
		if err != nil {
			return err
		}

		ok = true
		return nil
	})

	if err != nil {
		return false, err
	}

	return ok, nil
}

func (d *Drive) GetRoot() (string, int64, error) {
	db, err := openBoltDB(d.dbPath)
	if err != nil {
		return "", -1, err
	}
	defer db.Close()

	var root string
	var rootVersion int64

	err = db.View(func(tx *bolt.Tx) error {
		metaDataBucket := tx.Bucket([]byte(MetaDataBucketName))
		root = string(metaDataBucket.Get([]byte("rootval")))
		rootVersion, err = strconv.ParseInt(string(metaDataBucket.Get([]byte("rootversion"))), 10, 64)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", -1, err
	}

	return root, rootVersion, nil
}

func (d *Drive) AddPack(name string, gcGeneration int64) (bool, error) {
	return false, errors.New("unimplemented")
}

func (d *Drive) RemovePack(name string, gcGeneration int64) (bool, error) {
	return false, errors.New("unimplemented")
}

func (d *Drive) GetPacks() ([]string, error) {
	return nil, errors.New("unimplemented")
}
