package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"p2p-music/internal/song"
	"strings"

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
			logger.Error("Failed to delete bucket", "bucket", songsBucket, "err", err)
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

func (s *Storage) AddSong(ctx context.Context, song song.Song) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(songsBucket))

		songBytes, err := json.Marshal(song)
		if err != nil {
			return err
		}

		return b.Put([]byte(song.Title), songBytes)
	})
}

func (s *Storage) CreateSongsList(ctx context.Context, songs []song.Song) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		for _, song := range songs {
			b := tx.Bucket([]byte(songsBucket))

			songBytes, err := json.Marshal(song)
			if err != nil {
				return err
			}

			if err := b.Put([]byte(song.Title), songBytes); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Storage) GetSongsList(ctx context.Context) ([]song.Song, error) {
	var songs []song.Song

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(songsBucket))

		return b.ForEach(func(k, v []byte) error {
			var song song.Song
			if err := json.Unmarshal(v, &song); err != nil {
				return err
			}

			songs = append(songs, song)
			return nil
		})
	})

	s.logger.Info("The amount of songs in the bucket", "number", len(songs))

	return songs, err
}

func (s *Storage) FindSongByTitle(ctx context.Context, title string) (song.Song, error) {
	var song song.Song

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(songsBucket))

		return b.ForEach(func(k, v []byte) error {
			if string(k) == title {
				if err := json.Unmarshal(v, &song); err != nil {
					s.logger.Error("Failed to unmarshal found song", "err", err)
				}
			}

			return nil
		})
	})

	// s.logger.Info("The amount of songs for title", "title", title, "amount", len(songs))

	return song, err
}

func (s *Storage) FindSongByCID(ctx context.Context, cid cid.Cid) (song.Song, error) {
	var sng song.Song

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(songsBucket))

		return b.ForEach(func(k, v []byte) error {
			var vSong song.Song
			if err := json.Unmarshal(v, &vSong); err != nil {
				s.logger.Error("Failed to unmarshal found song", "err", err)
				return nil
			}

			if vSong.CID.Equals(cid) {
				sng = vSong
			}

			return nil
		})
	})

	if (sng == song.Song{}) {
		return song.Song{}, fmt.Errorf("couldn't find song by CID: %s", cid.String())
	}

	return sng, err
}

// TODO: think about cuncurrent search :: search data by uploading batches of songs in-memory
func (s *Storage) FindSongsByTitle(ctx context.Context, title string) ([]song.Song, error) {
	var songs []song.Song

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(songsBucket))

		return b.ForEach(func(k, v []byte) error {
			if strings.ContainsAny(string(k), title) {
				var song song.Song
				if err := json.Unmarshal(v, &song); err != nil {
					s.logger.Error("Failed to unmarshal found song", "err", err)
				}
				songs = append(songs, song)
			}

			return nil
		})
	})

	s.logger.Info("The amount of songs for title", "title", title, "amount", len(songs))

	return songs, err
}

// TODO: implement:)
func (s *Storage) FindSongsWithParams(context.Context, song.Song) ([]song.Song, error) {
	return nil, nil
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
