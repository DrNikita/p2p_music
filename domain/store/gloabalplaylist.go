package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"strings"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
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

	go p.streamListenerLoop()
	return p, nil
}

func (p *GlobalPlaylist) RegisterGetPlaylistHandler(ctx context.Context, h host.Host) {
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

func (p *GlobalPlaylist) AdvertiseSong(song Song) error {
	songBytes, err := json.Marshal(song)
	if err != nil {
		return err
	}

	p.Songs = append(p.Songs, song)

	return p.topic.Publish(p.ctx, songBytes)
}

func (p *GlobalPlaylist) Search(songName string) (Song, error) {
	for _, song := range p.Songs {
		if strings.ContainsAny(song.Title, songName) {
			return song, nil
		}
	}
	return Song{}, fmt.Errorf("failed to find song for provided name %s", songName)
}

func (p *GlobalPlaylist) streamListenerLoop() {
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
