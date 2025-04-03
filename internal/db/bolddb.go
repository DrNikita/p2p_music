package db

import (
	"context"
	"log"
	"log/slog"
	"time"

	"github.com/boltdb/bolt"
	"github.com/ipfs/go-cid"
)

const (
	dbFile     = "cid_store.db"
	bucketName = "cid_to_path"
)

type Storage struct {
	db     *bolt.DB
	logger *slog.Logger
}

func InitDB(logger *slog.Logger) (*Storage, error) {
	boltDB, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer boltDB.Close()

	err = boltDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &Storage{
		db:     boltDB,
		logger: logger,
	}, nil
}

func (s *Storage) SaveFilePath(ctx context.Context, CID cid.Cid, path string) error {
	s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		return b.Put(CID.Bytes(), []byte(path))
	})
	return nil
}

func (s *Storage) FindFilePath(ctx context.Context, CID cid.Cid) (string, error) {
	var path string

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		val := b.Get(CID.Bytes())
		if val == nil {
			return nil
		}

		path = string(val)

		return nil
	})

	return path, err
}
