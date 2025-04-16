package song

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/ipfs/go-cid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type SongTableStore interface {
	GetSongsList(context.Context) ([]Song, error)

	FindSongsByTitle(context.Context, string) ([]Song, error)

	FindSongByTitle(ctx context.Context, title string) (Song, error)

	FindSongByCID(ctx context.Context, cid cid.Cid) (Song, error)

	FindSongsWithParams(context.Context, Song) ([]Song, error)

	AddSong(context.Context, Song) (Song, error)

	CreateSongsList(context.Context, []Song) error
}

const (
	songTableTopic       = "song_table"
	getSongTableProtocol = "/songtable/get/1.0.0"
)

type SongTableSync struct {
	ctx    context.Context
	ps     *pubsub.PubSub
	topic  *pubsub.Topic
	sub    *pubsub.Subscription
	self   peer.ID
	logger *slog.Logger

	songTableStore SongTableStore
}

func SetupSongTableSync(ctx context.Context, h host.Host, songTableStore SongTableStore, logger *slog.Logger) (*SongTableSync, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		logger.Error("Failed to create gossipsub", "err", err)
		return nil, err
	}

	topic, err := ps.Join(songTableTopic)
	if err != nil {
		logger.Error("Gossip sub join failure", "topic name", songTableStore, "err", err)
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		logger.Error("Subscription failure", "topic name", songTableStore, "err", err)
		return nil, err
	}

	// ticker := time.NewTicker(time.Millisecond * 100)
	// defer ticker.Stop()
	// var peers []peer.ID

	// for len(peers) == 0 {
	// 	select {
	// 	case <-ctx.Done():
	// 		return nil, fmt.Errorf("timed out waiting for peers in topic")
	// 	case <-ticker.C:
	// 		peers = append(peers, topic.ListPeers()...)
	// 	}
	// }

	//TODO: think about how to get rid of Sleep func
	time.Sleep(time.Second)

	//TODO: implement re-reveiving songs
	songs, err := receiveSongs(ctx, h, logger)
	if err != nil {
		logger.Error("Failed to receive songs", "err", err)
		return nil, err
	}

	logger.Info("Received songs", "=======songs_count========", len(songs))

	if err := songTableStore.CreateSongsList(ctx, songs); err != nil {
		return nil, err
	}

	p := &SongTableSync{
		ctx:    ctx,
		ps:     ps,
		topic:  topic,
		sub:    sub,
		self:   h.ID(),
		logger: logger,

		songTableStore: songTableStore,
	}

	songsChan := p.streamListenerLoop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case song := <-songsChan:
				logger.Info("New song received", "song title", song.Title)
				if _, err := songTableStore.AddSong(ctx, *song); err != nil {
					logger.Error("Failed to add song to BoltDB", "err", err)
				}
			}
		}
	}()

	return p, nil
}

func (ts *SongTableSync) RegisterSongTableHandlers(ctx context.Context, h host.Host) {
	h.SetStreamHandler(getSongTableProtocol, ts.sendSongsToStream)
}

func (ts *SongTableSync) AdvertiseSong(song Song) error {
	songBytes, err := json.Marshal(song)
	if err != nil {
		return err
	}

	return ts.topic.Publish(ts.ctx, songBytes)
}

func (ts *SongTableSync) sendSongsToStream(s network.Stream) {
	defer s.Close()

	songs, err := ts.songTableStore.GetSongsList(ts.ctx)
	if err != nil {
		ts.logger.Error("Failed to get songs from BoltDB", "err", err)
		return
	}
	if songs == nil {
		ts.logger.Info("Epmty songs list")
		return
	}

	songsByte, err := json.Marshal(songs)
	if err != nil {
		s.Reset()
		return
	}

	_, err = s.Write(songsByte)
	if err != nil {
		ts.logger.Error("Failed to write song bytes to stream", "err", err)
	}
}

func receiveSongs(ctx context.Context, h host.Host, logger *slog.Logger) ([]Song, error) {
	pHolder := getSongTableHolder(h)
	if pHolder == "" {
		return make([]Song, 0), nil
	}

	s, err := h.NewStream(ctx, pHolder, getSongTableProtocol)
	if err != nil {
		logger.Error("Failed to create new stream", "err", err)
		return nil, err
	}
	defer s.Close()

	songsBytes, err := io.ReadAll(s)
	if err != nil {
		logger.Error("Failed to read from stream", "err", err)
		return nil, err
	}
	if len(songsBytes) == 0 {
		logger.Info("Empty songs bytes received")
		// TODO: return err
		return nil, nil
	}

	var songs []Song
	err = json.Unmarshal(songsBytes, &songs)
	if err != nil {
		logger.Error("Failed to unmarshal songs", "err", err)
		return nil, err
	}

	return songs, nil
}

func (ts *SongTableSync) streamListenerLoop() <-chan *Song {
	songsChan := make(chan *Song)

	go func() {
		defer close(songsChan)
		for {
			select {
			case <-ts.ctx.Done():
				return
			default:
				msg, err := ts.sub.Next(ts.ctx)
				if err != nil {
					return
				}

				// only forward messages delivered by others
				if msg.ReceivedFrom == ts.self {
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
func (ts *SongTableSync) ListPeers() []peer.ID {
	return ts.ps.ListPeers(songTableTopic)
}
