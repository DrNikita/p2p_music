package peerdiscovery

import (
	"context"
	"fmt"
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

func Bootstrap(

	ctx context.Context,

	h host.Host,

	bootstrapPeers []multiaddr.Multiaddr,

	configs *config.Config,

	logger *slog.Logger,

) (func() error, song.SongTableSynchronizer) {
	// Peer discovery
	peerDiscoverer := NewDHTManager(h, logger)
	kdht, err := peerDiscoverer.NewDHT(ctx, bootstrapPeers)
	if err != nil {
		logger.Error("Error creating KAD", "err", err)
		log.Fatal(err)
	}

	go peerDiscoverer.Discover(ctx, kdht, nodeNamespace)

	store, closeDBConn, err := db.InitDB(logger)
	if err != nil {
		logger.Error("Failed to init BoltDB", "err", err)
		log.Fatal(err)
	}

	// Global song list initialization
	songTable, err := song.SetupSongTableSync(ctx, h, store, logger)
	if err != nil {
		logger.Error("Setup global palylist error", "err", err)
		log.Fatal(err)
	}
	songTable.RegisterSongTableHandlers(ctx, h)

	songTableManager := song.NewSongManager(h, songTable, kdht, store, store, configs, logger)
	songTableManager.RegisterSongStreamingProtocols(ctx)

	////////////////////
	//.....TESTING......
	if len(bootstrapPeers) == 0 {
		select {}
	}

	song, err := song.NewSong(configs.TestFilePath)
	if err != nil {
		log.Fatal(err)
	}

	err = songTableManager.PromoteSong(ctx, song, configs.TestFilePath)
	if err != nil {
		log.Fatal(err)
	}

	songProviders, err := songTableManager.FindSongProviders(ctx, song)
	if err != nil {
		log.Fatal(err)
	}

	if len(songProviders) > 0 {
		receivedSongFilePath, err := songTableManager.ReceiveSongStream(ctx, song, songProviders[len(songProviders)-1].ID)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("_____________________________", receivedSongFilePath)

		// if err := songTableManager.PromoteSong(ctx, song, receivedSongFilePath); err != nil {
		// 	log.Fatal(err)
		// }
	}
	// //////////////////
	// .....TESTING......

	return closeDBConn, songTable
}
