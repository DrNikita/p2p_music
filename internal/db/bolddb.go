package db

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
)

const (
	bucketName = "cid_to_path"
)

type Storage struct {
	db     *bolt.DB
	logger *slog.Logger
}

func InitDB(logger *slog.Logger) (*Storage, func() error, error) {
	uuid := uuid.New()

	dbFile := fmt.Sprintf("cid_store_%s.db", uuid.String())
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	closeDBConn := func() error {
		if err := db.Close(); err != nil {
			return err
		}
		return nil
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		return nil, closeDBConn, err
	}

	return &Storage{
		db:     db,
		logger: logger,
	}, closeDBConn, nil
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

	s.logger.Info("File path for song", "CID", CID, "value", path)

	return path, err
}
