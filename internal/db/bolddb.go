package db

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"p2p-music/internal/song"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
)

const (
	pathsBucket = "cid_to_path"
	songsBucket = "songs_metadata"
)

type Storage struct {
	db     *bolt.DB
	logger *slog.Logger
}

func InitDB(logger *slog.Logger) (*Storage, func() error, error) {
	uuid := uuid.New()

	//TODO: const db name
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

	// Bucket fo paths
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(pathsBucket))
		return err
	})
	if err != nil {
		return nil, closeDBConn, err
	}

	// Bucket for songs
	err = db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket([]byte(songsBucket)); err != nil {
			return err
		}

		_, err := tx.CreateBucketIfNotExists([]byte(songsBucket))
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

/*
	type SongTableManager interface {
		GetSongsList(ctx context.Context) ([]song.Song, error)

		FindSong(ctx context.Context) (song.Song, error)

		FindSongWithParams(ctx context.Context) (song.Song, error)

		AddSong(ctx context.Context, song song.Song) error
	}
*/

func (s *Storage) GetSongsList(ctx context.Context) ([]song.Song, error) {
	return nil, nil
}
func (s *Storage) FindSong(ctx context.Context) (song.Song, error) {
	return song.Song{}, nil
}
func (s *Storage) FindSongWithParams(ctx context.Context) (song.Song, error) {
	return song.Song{}, nil
}
func (s *Storage) AddSong(ctx context.Context, song song.Song) error {
	return nil
}
func (s *Storage) CreateSongList(ctx context.Context, songs []song.Song) error {
	return nil
}

func (s *Storage) SaveFilePath(ctx context.Context, CID cid.Cid, path string) error {
	s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(pathsBucket))
		return b.Put(CID.Bytes(), []byte(path))
	})
	return nil
}

func (s *Storage) FindFilePath(ctx context.Context, CID cid.Cid) (string, error) {
	var path string

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(pathsBucket))
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
