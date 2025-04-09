package peerdiscovery

import (
	"context"
	"log"
	"log/slog"
	"p2p-music/config"
	"p2p-music/internal/db"
	"p2p-music/internal/song"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
)

const (
	nodeNamespace string = "music"
)

type PeerDiscoverer interface {
	NewDHT(ctx context.Context, bootstrapPeers []multiaddr.Multiaddr) (*dht.IpfsDHT, error)
	Discover(ctx context.Context, kdht *dht.IpfsDHT, rendezvous string)
}

func Bootstrap(ctx context.Context, h host.Host, bootstrapPeers []multiaddr.Multiaddr, configs *config.Config, logger *slog.Logger) func() error {
	// Peer discovery
	peerDiscoverer := NewDHTManager(h, logger)
	kdht, err := peerDiscoverer.NewDHT(ctx, bootstrapPeers)
	if err != nil {
		logger.Error("Error creating KAD", "err", err)
		log.Fatal(err)
	}

	go peerDiscoverer.Discover(ctx, kdht, nodeNamespace)

	// Global song list initialization
	songTable, err := song.SetupSongTable(ctx, h, logger)
	if err != nil {
		logger.Error("Setup global palylist error", "err", err)
		log.Fatal(err)
	}
	songTable.RegisterSongTableHandlers(ctx, h)

	store, closeDBConn, err := db.InitDB(logger)
	if err != nil {
		logger.Error("Failed to init BoltDB", "err", err)
		log.Fatal(err)
	}

	songTableManager := song.NewSongManager(h, songTable, kdht, store, configs, logger)
	songTableManager.RegisterSongStreamingProtocols(ctx)

	return closeDBConn
}
