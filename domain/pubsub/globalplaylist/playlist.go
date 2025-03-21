package globalplaylist

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	PlaylistBufSize     = 128
	GlobalPlaylistTopic = "global_playlist"
)

type GlobalPlaylist struct {
	Songs chan *Song

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

// mb Murshal/Unmarshal are will be needed
// mb compare songs before adding

func SubscribeToGlobalPlaylist(ctx context.Context, ps *pubsub.PubSub, self peer.ID) (*GlobalPlaylist, error) {
	topic, err := ps.Join(GlobalPlaylistTopic)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	p := &GlobalPlaylist{
		ctx:   ctx,
		ps:    ps,
		topic: topic,
		sub:   sub,
		self:  self,
		Songs: make(chan *Song, PlaylistBufSize),
	}

	go p.readLoop()

	return p, nil
}

func (p *GlobalPlaylist) AdvertiseSong(song Song) error {
	songBytes, err := json.Marshal(song)
	if err != nil {
		return err
	}

	return p.topic.Publish(p.ctx, songBytes)
}

func (p *GlobalPlaylist) ListPeers() []peer.ID {
	return p.ps.ListPeers(GlobalPlaylistTopic)
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (p *GlobalPlaylist) readLoop() {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			msg, err := p.sub.Next(p.ctx)
			if err != nil {
				close(p.Songs)
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

			p.Songs <- song

			fmt.Println("__________NEW MSG____________", song.Album)
		}
	}
}
