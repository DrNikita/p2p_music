package song

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

//TODO: interface needed

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

	cid, err := GenerateSongCID(file)
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

func GenerateSongCID(song *os.File) (cid.Cid, error) {
	hasher := sha256.New()

	// Stream the file content through the hasher
	if _, err := io.Copy(hasher, song); err != nil {
		return cid.Undef, fmt.Errorf("failed to hash file: %w", err)
	}

	// Encode the hash as a multihash
	mh, err := multihash.Encode(hasher.Sum(nil), multihash.SHA2_256)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to encode multihash: %w", err)
	}

	return cid.NewCidV1(cid.Raw, mh), nil
}

func (s Song) SongNameWithoutFormat() string {
	titleParts := strings.Split(s.Title, ".")
	if len(titleParts) < 1 {
		return uuid.NewString()
	}

	noFormatPart := titleParts[len(titleParts)-2]
	noFormatPartSpleted := strings.Split(noFormatPart, "/")
	if len(noFormatPartSpleted) == 0 {
		return uuid.NewString()
	}

	return noFormatPartSpleted[len(noFormatPartSpleted)-1]
}

func (s Song) SongFormat() string {
	titleParts := strings.Split(s.Title, ".")
	if len(titleParts) < 1 {
		return uuid.NewString()
	}

	return titleParts[len(titleParts)-1]
}
