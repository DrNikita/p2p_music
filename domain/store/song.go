package store

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

type Song struct {
	Title    string
	Artist   string
	Album    string
	Year     int
	Format   string
	Bitrate  int
	FileSize int64
	Duration time.Duration
	CID      cid.Cid
}

func NewSong(filePath string) (Song, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Song{}, err
	}
	defer file.Close()

	cid, err := GenerateCIDFromFile(file)
	if err != nil {
		return Song{}, err
	}

	var format string
	switch {
	case strings.HasSuffix(filePath, "mp3"):
		format = "mp3"
	case strings.HasSuffix(filePath, "ogg"):
		format = "ogg"
	}

	return Song{
		Title:  file.Name(),
		Format: format,
		CID:    cid,
	}, nil
}

func GenerateCIDFromFile(file *os.File) (cid.Cid, error) {
	// Create a SHA-256 hasher
	hasher := sha256.New()

	// Stream the file content through the hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return cid.Undef, fmt.Errorf("failed to hash file: %w", err)
	}

	// Encode the hash as a multihash
	mh, err := multihash.Encode(hasher.Sum(nil), multihash.SHA2_256)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to encode multihash: %w", err)
	}

	// Create a CID using the multihash
	return cid.NewCidV1(cid.Raw, mh), nil
}
