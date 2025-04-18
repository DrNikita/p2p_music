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
	"github.com/hbollon/go-edlib"
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

	//TODO: const db name; It's tmp desicion for testing multiple app instances
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
		tx.DeleteBucket([]byte(songsBucket))

		if _, err := tx.CreateBucketIfNotExists([]byte(pathsBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(songsBucket)); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, closeDBConn, err
	}

	return &Storage{
		db:     db,
		logger: logger,
	}, closeDBConn, nil
}

func (s *Storage) AddSong(ctx context.Context, pSong song.Song) (song.Song, error) {
	var aSong song.Song

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(songsBucket))

		if existingSong, ok := s.checkSongIfExists(ctx, pSong); ok {
			aSong = existingSong
			return nil
		}

		songBytes, err := json.Marshal(pSong)
		if err != nil {
			return err
		}
		aSong = pSong

		return b.Put([]byte(pSong.Title), songBytes)
	})
	if err != nil {
		return song.Song{}, err
	}

	return aSong, nil
}

// func (s *Storage) UpdateSong(ctx context.Context, song song.Song) error {
// 	return nil
// }

func (s *Storage) CreateSongsList(ctx context.Context, songs []song.Song) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		for _, song := range songs {
			b := tx.Bucket([]byte(songsBucket))

			if existing := b.Get([]byte(song.Title)); existing != nil {
				s.logger.Warn("Song already exists", "title", song.Title)
				continue
			}

			songBytes, err := json.Marshal(song)
			if err != nil {
				s.logger.Error("Failed to marshal song", "err", err)
				return err
			}

			if err := b.Put([]byte(song.Title), songBytes); err != nil {
				s.logger.Error("Failed to put song", "err", err)
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
	var songFound song.Song

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(songsBucket))

		return b.ForEach(func(k, v []byte) error {
			if string(k) == title {
				if err := json.Unmarshal(v, &songFound); err != nil {
					s.logger.Error("Failed to unmarshal found song", "err", err)
				}
			}

			return nil
		})
	})

	if (songFound == song.Song{}) {
		return song.Song{}, errSongNotFound
	}

	return songFound, err
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

func (s *Storage) checkSongIfExists(ctx context.Context, songParam song.Song) (song.Song, bool) {
	var existingSong song.Song

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(songsBucket))

		return b.ForEach(func(k, v []byte) error {
			var vSong song.Song
			if err := json.Unmarshal(v, &vSong); err != nil {
				s.logger.Error("Failed to unmarshal found song", "err", err)
			}

			match, err := edlib.StringsSimilarity(string(k), songParam.Title, edlib.JaroWinkler)
			if err != nil {
				s.logger.Error("Failed to compare songs titles", "err", err)
			}

			if match > 0.8 {
				existingSong = vSong
				return nil
			} else if vSong.CID == songParam.CID {
				existingSong = vSong
				return nil
			}

			return nil
		})
	})
	if err != nil || existingSong.Title == "" || (existingSong.CID == cid.Undef) {
		return existingSong, false
	}

	return existingSong, true
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
