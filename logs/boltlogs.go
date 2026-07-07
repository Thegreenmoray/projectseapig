package logs

import (
	"encoding/json"
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

func (r *BoltRepo) SavePig(testName string, pig runners.Pig) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		// 1. Ensure the root bucket for tests exists
		bucket, err := tx.CreateBucketIfNotExists([]byte("TestHistory"))
		if err != nil {
			return err
		}

		// 2. Serialize your Pig struct into bytes (JSON works great here)
		pigBytes, err := json.Marshal(pig)
		if err != nil {
			return err
		}

		// 3. Create a unique key (combining test name and timestamp)
		key := testName + "_" + time.Now().Format(time.RFC3339)

		// 4. Save it!
		return bucket.Put([]byte(key), pigBytes)
	})
}
