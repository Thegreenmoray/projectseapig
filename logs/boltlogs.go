package logs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Justi/projectseapig/runners"
	"go.etcd.io/bbolt"
)

type BoltRepo struct {
	db *bbolt.DB
}

func NewBoltRepo(dbPath string) (*BoltRepo, error) {
	// Opens the database file (creates it if it doesn't exist)
	// 0600 gives read/write permissions only to the owner
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	return &BoltRepo{db: db}, nil
}

func (r *BoltRepo) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *BoltRepo) SavePig(testName string, pig *runners.Pig) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("TestHistory"))
		if err != nil {
			return err
		}

		// Preserve existing timestamp or set fallback
		if pig.Dateandtime == "" {
			pig.Dateandtime = time.Now().Format(time.RFC3339)
		}

		pigBytes, err := json.Marshal(pig)
		if err != nil {
			return err
		}

		// Unique, sortable key using UnixNano
		key := fmt.Sprintf("%s_%d", testName, time.Now().UnixNano())

		return bucket.Put([]byte(key), pigBytes)
	})
}
