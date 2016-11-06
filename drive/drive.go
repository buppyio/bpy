package drive

import (
	"errors"
	"github.com/boltdb/bolt"
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
		_, err := tx.CreateBucketIfNotExists([]byte(MetaDataBucketName))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(PacksBucketName))
		if err != nil {
			return err
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
	return 0, errors.New("unimplemented")
}

func (d *Drive) StartGC() error {
	return errors.New("unimplemented")
}

func (d *Drive) StopGC() error {
	return errors.New("unimplemented")
}

func (d *Drive) CasRoot(root string, newVersion, gcGeneration int64) (bool, error) {
	return false, errors.New("unimplemented")
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
