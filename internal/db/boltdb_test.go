package db

import (
	"context"
	"errors"
	"log/slog"
	"p2p-music/internal/song"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/ipfs/go-cid"

	"github.com/stretchr/testify/require"
)

const (
	testDBName = "boltdb_test.db"
)

func MustOpenDB() *Storage {
	db, err := bolt.Open(testDBName, 0666, nil)
	if err != nil {
		panic(err)
	}

	return &Storage{
		db:     db,
		logger: slog.Default(),
	}
}

func (s *Storage) MustClose() {
	if err := s.db.Close(); err != nil {
		panic(err)
	}
}

func (s *Storage) createBuckets() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		tx.DeleteBucket([]byte(songsBucket))

		if _, err := tx.CreateBucketIfNotExists([]byte(pathsBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(songsBucket)); err != nil {
			return err
		}

		return nil
	})
}

func (s *Storage) deleteBuckets() {
	s.db.Update(func(tx *bolt.Tx) error {
		tx.DeleteBucket([]byte(pathsBucket))
		tx.DeleteBucket([]byte(songsBucket))
		return nil
	})
}

func TestAddSong(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())

	db := MustOpenDB()
	err := db.createBuckets()
	require.NoError(t, err)

	defer func() {
		db.deleteBuckets()
		db.MustClose()
	}()

	dummyCid, err := cid.Parse("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name     string
		song     song.Song
		testFunc func(song.Song) error
		wantErr  error
	}{
		{
			name: "1. AddSong: success",
			song: song.Song{
				Title: "test_title",
				CID:   dummyCid,
			},
			testFunc: func(song song.Song) error {
				if err := db.AddSong(ctx, song); err != nil {
					return err
				}
				return nil
			},
			wantErr: nil,
		},
		{
			name: "2. AddSong: success",
			song: song.Song{
				Title: "test_title",
				CID:   dummyCid,
			},
			testFunc: func(song song.Song) error {
				if err := db.AddSong(ctx, song); err != nil {
					return err
				}
				return errors.New("anything")
			},
			wantErr: errors.New("anything"),
		},
	}

	for _, tc := range testCases {
		if tc.testFunc == nil {
			continue
		}

		t.Run(tc.name, func(t *testing.T) {
			err := tc.testFunc(tc.song)
			if tc.wantErr != nil {
				require.EqualError(t, err, tc.wantErr.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
