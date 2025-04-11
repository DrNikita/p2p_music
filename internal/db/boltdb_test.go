package db

import (
	"p2p-music/internal/song"
	"testing"

	"github.com/boltdb/bolt"
)

func MustOpenDB() *Storage {
	db, err := bolt.Open("./tmpdb", 0666, nil)
	if err != nil {
		panic(err)
	}
	return &Storage{
		db: db,
	}
}

func (db *Storage) MustClose() {
	if err := db.db.Close(); err != nil {
		panic(err)
	}
}

func TestConsistency(t *testing.T) {
	db := MustOpenDB()
	defer db.MustClose()
	db.AddSong(nil, song.Song{})
	//...
}
