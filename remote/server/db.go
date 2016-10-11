package server

import (
	"encoding/json"
	"github.com/boltdb/bolt"
	"time"
)

const (
	RefBucketName     = "refs"
	KeyIdBucketName   = "keyid"
	GCStateBucketName = "gc"
	BpyDBName         = "bpy.db"
)

type refListingFile struct {
	keyId     string
	offset    uint64
	refDBPath string
	entries   []string
}

type gcState struct {
	Generation uint64
	ID         string
}

func getGCState(tx *bolt.Tx) (gcState, error) {
	stateBucket := tx.Bucket([]byte(GCStateBucketName))
	stateBytes := stateBucket.Get([]byte("state"))
	state := gcState{}
	err := json.Unmarshal(stateBytes, &state)
	if err != nil {
		return state, err
	}
	return state, err
}

func setGCState(tx *bolt.Tx, state gcState) error {
	stateBucket := tx.Bucket([]byte(GCStateBucketName))
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return stateBucket.Put([]byte("state"), data)
}

func openDB(dbPath, keyId string) (*bolt.DB, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(RefBucketName))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(KeyIdBucketName))
		if err != nil {
			return err
		}
		gcStateBucket, err := tx.CreateBucketIfNotExists([]byte(GCStateBucketName))
		if err != nil {
			return err
		}
		if gcStateBucket.Get([]byte("state")) == nil {
			err = setGCState(tx, gcState{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
