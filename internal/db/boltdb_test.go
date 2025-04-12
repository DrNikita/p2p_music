package db

import (
	"context"
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
	db, err := bolt.Open(testDBName, 0600, nil)
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
	defer db.MustClose()

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
				Title: "test_title_1",
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
			name: "2. AddSong: error: duplicate key",
			song: song.Song{
				Title: "test_title_1",
				CID:   dummyCid,
			},
			testFunc: func(song song.Song) error {
				if err := db.AddSong(ctx, song); err != nil {
					return err
				}
				if err := db.AddSong(ctx, song); err != nil {
					return err
				}
				return nil
			},
			wantErr: errDuplicateKey,
		},
	}

	for _, tc := range testCases {
		if tc.testFunc == nil {
			continue
		}

		err := db.createBuckets()
		require.NoError(t, err)

		t.Run(tc.name, func(t *testing.T) {
			err := tc.testFunc(tc.song)
			if tc.wantErr != nil {
				require.EqualError(t, err, tc.wantErr.Error())
				return
			}
			require.NoError(t, err)
		})

		db.deleteBuckets()
	}
}

func TestCreateSongsList(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())

	db := MustOpenDB()
	defer db.MustClose()

	dummyCid, err := cid.Parse("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name     string
		songs    []song.Song
		testFunc func([]song.Song) error
		wantErr  error
	}{
		{
			name: "1. CreateSongsList: success",
			songs: []song.Song{
				{
					Title: "test_title_1",
					CID:   dummyCid,
				},
				{
					Title: "test_title_2",
					CID:   dummyCid,
				},
				{
					Title: "test_title_3",
					CID:   dummyCid,
				},
			},
			testFunc: func(songs []song.Song) error {
				if err := db.CreateSongsList(ctx, songs); err != nil {
					return err
				}
				return nil
			},
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		if tc.testFunc == nil {
			continue
		}

		err := db.createBuckets()
		require.NoError(t, err)

		t.Run(tc.name, func(t *testing.T) {
			err := tc.testFunc(tc.songs)
			if tc.wantErr != nil {
				require.EqualError(t, err, tc.wantErr.Error())
				return
			}
			require.NoError(t, err)
		})

		db.deleteBuckets()
	}
}

func TestFindSongByTitle(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())

	db := MustOpenDB()
	defer db.MustClose()

	dummyCid, err := cid.Parse("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name     string
		song     song.Song
		testFunc func(song.Song) (song.Song, error)
		wantErr  error
	}{
		{
			name: "1. FindSongByTitle: success",
			song: song.Song{
				Title: "test_title_1",
				CID:   dummyCid,
			},
			testFunc: func(s song.Song) (song.Song, error) {
				if err := db.AddSong(ctx, s); err != nil {
					return song.Song{}, err
				}

				s, err := db.FindSongByTitle(ctx, s.Title)
				if err != nil {
					return s, err
				}
				return s, nil
			},
			wantErr: nil,
		},
		{
			name: "2. FindSongByTitle: failure: not found",
			song: song.Song{
				Title: "test_title_2",
				CID:   dummyCid,
			},
			testFunc: func(s song.Song) (song.Song, error) {
				s, err := db.FindSongByTitle(ctx, s.Title)
				if err != nil {
					return s, err
				}
				return s, nil
			},
			wantErr: errSongNotFound,
		},
	}

	for _, tc := range testCases {
		if tc.testFunc == nil {
			continue
		}

		err := db.createBuckets()
		require.NoError(t, err)

		t.Run(tc.name, func(t *testing.T) {
			song, err := tc.testFunc(tc.song)
			if tc.wantErr != nil {
				require.EqualError(t, err, tc.wantErr.Error())
				return
			}

			require.NoError(t, err)

			require.Equal(t, tc.song, song)
		})

		db.deleteBuckets()
	}
}

func TestGetSongsList(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())

	db := MustOpenDB()
	defer db.MustClose()

	dummyCid, err := cid.Parse("QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name     string
		songs    []song.Song
		testFunc func([]song.Song) ([]song.Song, error)
		wantErr  error
	}{
		{
			name: "1. GetSongsList: success",
			songs: []song.Song{
				{
					Title: "test_title_1",
					CID:   dummyCid,
				},
				{
					Title: "test_title_2",
					CID:   dummyCid,
				},
				{
					Title: "test_title_3",
					CID:   dummyCid,
				},
			},
			testFunc: func(s []song.Song) ([]song.Song, error) {
				if err := db.CreateSongsList(ctx, s); err != nil {
					return nil, err
				}

				songs, err := db.GetSongsList(ctx)
				if err != nil {
					return nil, err
				}
				return songs, nil
			},
			wantErr: nil,
		},
		{
			name: "2. GetSongsList: arrays not equal",
			songs: []song.Song{
				{
					Title: "test_title_1",
					CID:   dummyCid,
				},
				{
					Title: "test_title_2",
					CID:   dummyCid,
				},
				{
					Title: "test_title_3",
					CID:   dummyCid,
				},
			},
			testFunc: func(s []song.Song) ([]song.Song, error) {
				songs, err := db.GetSongsList(ctx)
				if err != nil {
					return nil, err
				}
				return songs, nil
			},
			wantErr: errTestArraysNotEqual,
		},
	}

	for _, tc := range testCases {
		if tc.testFunc == nil {
			continue
		}

		err := db.createBuckets()
		require.NoError(t, err)

		t.Run(tc.name, func(t *testing.T) {
			songs, err := tc.testFunc(tc.songs)
			if tc.wantErr != nil && tc.wantErr != errTestArraysNotEqual {
				require.EqualError(t, err, tc.wantErr.Error())
				return
			}

			require.NoError(t, err)

			if tc.wantErr == errTestArraysNotEqual {
				require.NotEqual(t, tc.songs, songs)
			} else {
				require.Equal(t, tc.songs, songs)
			}
		})

		db.deleteBuckets()
	}
}

func TestFindSongsByTitle(t *testing.T) {}
