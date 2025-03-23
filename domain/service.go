package domain

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

func generateCIDFromFile(filePath string) (cid.Cid, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

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

func getSongInfo() {}
