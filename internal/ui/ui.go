package domain

import (
	"context"
	"log/slog"
	"p2p-music/internal/song"

	"github.com/libp2p/go-libp2p/core/peer"
	// tea "github.com/charmbracelet/bubbletea"
)

type SongManagerInterface interface {
	PromoteSong(ctx context.Context, song song.Song) error

	RegisterSongStreamingProtocols(ctx context.Context)

	ReceiveSongStream(ctx context.Context, song song.Song, targetPeerID peer.ID) error

	FindSongProviders(ctx context.Context, song song.Song) ([]peer.AddrInfo, error)
}

type PeerInteractor struct {
	songManager SongManagerInterface
	logger      *slog.Logger
}

func NewUI(songManager SongManagerInterface, logger *slog.Logger) *PeerInteractor {
	return &PeerInteractor{
		songManager: songManager,
		logger:      logger,
	}
}

func (ui *PeerInteractor) F() {}
