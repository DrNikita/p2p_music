package song

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math/rand/v2"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type SongTableDB interface {
	GetSongsList(context.Context) ([]Song, error)

	FindSong(context.Context) (Song, error)

	FindSongWithParams(context.Context) (Song, error)

	AddSong(context.Context, Song) error

	CreateSongList(context.Context, []Song) error
}

const (
	songTableTopic       = "song_table"
	getSongTableProtocol = "/songtable/get/1.0.0"
)

type SongTable struct {
	ctx   context.Context
	ps    *pubsub.PubSub
	topic *pubsub.Topic
	sub   *pubsub.Subscription
	self  peer.ID

	songTableDB SongTableDB

	logger *slog.Logger
}

func SetupSongTable(ctx context.Context, h host.Host, songTableDB SongTableDB, logger *slog.Logger) (*SongTable, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		logger.Error("Failed to create gossipsub", "err", err)
		return nil, err
	}

	topic, err := ps.Join(songTableTopic)
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

	if err := songTableDB.CreateSongList(ctx, songs); err != nil {
		return nil, err
	}

	p := &SongTable{
		ctx:   ctx,
		ps:    ps,
		topic: topic,
		sub:   sub,
		self:  h.ID(),

		logger: logger,
	}

	songsChan := p.streamListenerLoop()
	go func() {
		for {
			song := <-songsChan
			if err := songTableDB.AddSong(ctx, *song); err != nil {
				logger.Error("Failed to add song to BoltDB", "err", err)
			}
		}
	}()

	return p, nil
}

func (p *SongTable) RegisterSongTableHandlers(ctx context.Context, h host.Host) {
	h.SetStreamHandler(getSongTableProtocol, p.sendSongsToStream)
}

func (p *SongTable) AdvertiseSong(song Song) error {
	songBytes, err := json.Marshal(song)
	if err != nil {
		return err
	}

	//TODO: mb it duplicates songs in SongTable
	if err := p.songTableDB.AddSong(p.ctx, song); err != nil {
		return err
	}

	return p.topic.Publish(p.ctx, songBytes)
}

func (p *SongTable) sendSongsToStream(s network.Stream) {
	defer s.Close()

	songs, err := p.songTableDB.GetSongsList(p.ctx)
	if err != nil {
		p.logger.Error("Failed to get songs from BoltDB", "err", err)
		return
	}

	songsByte, err := json.Marshal(songs)
	if err != nil {
		s.Reset()
		return
	}

	_, err = s.Write(songsByte)
	if err != nil {
		p.logger.Error("Failed to write song bytes to stream", "err", err)
	}
}

func receiveSongs(ctx context.Context, h host.Host) ([]Song, error) {
	pHolder := getSongTableHolder(h)
	if pHolder == "" {
		return make([]Song, 0), nil
	}

	s, err := h.NewStream(ctx, pHolder, getSongTableProtocol)
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

func (p *SongTable) streamListenerLoop() <-chan *Song {
	songsChan := make(chan *Song)

	go func() {
		defer close(songsChan)
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

				songsChan <- song
			}
		}
	}()

	return songsChan
}

// TODO: mb integrate method like iin ListPeers() func
func getSongTableHolder(h host.Host) peer.ID {
	peers := h.Network().Peers()
	if len(peers) == 0 {
		return ""
	}
	return peers[rand.IntN(len(peers))]
}

// TODO: needed?
func (p *SongTable) ListPeers() []peer.ID {
	return p.ps.ListPeers(songTableTopic)
}
