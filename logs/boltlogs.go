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

func (r *BoltRepo) SavePigtime(testName string, pig *runners.Pig) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("TestTime"))
		if err != nil {
			return err
		}

		// Preserve existing timestamp or set fallback
		if pig.Dateandtime == "" {
			pig.Dateandtime = time.Now().Format(time.RFC3339)
		}
		for i := 0; i < len(pig.Run); i++ {
			pigBytes, err := json.Marshal(pig.Run[i].Timetaken.Nanoseconds()) //just return the test results
			if err != nil {
				return err
			}
			key := fmt.Sprintf("%s_%d", testName, i)

			if err := bucket.Put([]byte(key), pigBytes); err != nil {
				return fmt.Errorf("cannot store data: %w", err)
			}

		}

		return nil
	})
}

func (r *BoltRepo) Extractpigtime() (map[string]int, error) {
	results := make(map[string]int)

	err := r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("TestTime"))
		if b == nil {
			return fmt.Errorf("TestTime does not exist")
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var durationNs int64
			if err := json.Unmarshal(v, &durationNs); err != nil {
				return fmt.Errorf("failed to parse test time for key %s: %w", string(k), err)
			}

			results[string(k)] = int(durationNs)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}
