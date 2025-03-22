package playlist

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"os"
	"time"

	"github.com/ipfs/go-cid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

const (
	globalPlaylistTopic       = "global_playlist"
	getGlobalPlaylistProtocol = "/playlist/get/1.0.0"
)

type GlobalPlaylist struct {
	Songs []Song

	ctx   context.Context
	ps    *pubsub.PubSub
	topic *pubsub.Topic
	sub   *pubsub.Subscription

	self peer.ID
}

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

func SetupGlobalPlaylist(ctx context.Context, ps *pubsub.PubSub, h host.Host, logger *slog.Logger) (*GlobalPlaylist, error) {
	topic, err := ps.Join(globalPlaylistTopic)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	//TODO: implement re-reveiving songs
	songs, err := receiveSongs(ctx, h)
	if err != nil {
		return nil, err
	}

	p := &GlobalPlaylist{
		ctx:   ctx,
		ps:    ps,
		topic: topic,
		sub:   sub,
		self:  h.ID(),
		Songs: songs,
	}

	go p.readLoop()

	return p, nil
}

func receiveSongs(ctx context.Context, h host.Host) ([]Song, error) {
	pHolder := getPlaylistHolder(h)
	if pHolder == "" {
		return make([]Song, 0), nil
	}

	s, err := h.NewStream(ctx, pHolder, getGlobalPlaylistProtocol)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	songsBytes, err := io.ReadAll(s)
	if err != nil {
		return nil, err
	}

	var songs []Song
	err = json.Unmarshal(songsBytes, &songs)
	if err != nil {
		return nil, err
	}

	return songs, nil
}

func (p *GlobalPlaylist) RegisterStreamHandlers(ctx context.Context, h host.Host) {
	h.SetStreamHandler(getGlobalPlaylistProtocol, p.sendSongsToStream)
}

func (p *GlobalPlaylist) sendSongsToStream(s network.Stream) {
	defer s.Close()

	songsByte, err := json.Marshal(p.Songs)
	if err != nil {
		s.Reset()
		return
	}

	_, err = s.Write(songsByte)
	if err != nil {
		//TODO: log
	}
}

func (p *GlobalPlaylist) AdvertiseSong(song Song) error {
	songBytes, err := json.Marshal(song)
	if err != nil {
		return err
	}

	return p.topic.Publish(p.ctx, songBytes)
}

func (p *GlobalPlaylist) readLoop() {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			msg, err := p.sub.Next(p.ctx)
			if err != nil {
				return
			}

			// only forward messages delivered by others
			if msg.ReceivedFrom == p.self {
				continue
			}

			song := new(Song)
			err = json.Unmarshal(msg.Data, song)
			if err != nil {
				continue
			}

			p.Songs = append(p.Songs, *song)
		}
	}
}

func getPlaylistHolder(h host.Host) peer.ID {
	peers := h.Network().Peers()
	if len(peers) == 0 {
		return ""
	}

	return peers[rand.IntN(len(peers))]

}

func (p *GlobalPlaylist) ListPeers() []peer.ID {
	return p.ps.ListPeers(globalPlaylistTopic)
}

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

// func (ds DiscoveryService) ProvideSong(ctx context.Context, filePath string) {
// 	contentID, err := generateCIDFromFile(filePath)
// 	if err := ds.kdht.Provide(ctx, contentID, true); err != nil {
// 		ds.logger.Error("Failed to provide content", "err", err)
// 		return
// 	}

// 	providers, err := ds.kdht.FindProviders(ctx, contentID)
// 	if err != nil {
// 		ds.logger.Error("Failed to find providers", "err", err)
// 		return
// 	}
// 	ds.logger.Info("Providers found", "count", len(providers), "providers", providers)
// }
