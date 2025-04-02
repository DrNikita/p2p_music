package domain

import (
	"context"
	"log"
	"log/slog"
	"p2p-music/config"
	"p2p-music/internal/domain/song"
	"p2p-music/internal/peerdiscovery"

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

func Run(ctx context.Context, h host.Host, bootstrapPeers []multiaddr.Multiaddr, configs *config.Config, logger *slog.Logger) {
	peerDiscoverer := peerdiscovery.NewDHTManager(h, logger)
	kdht, err := peerDiscoverer.NewDHT(ctx, bootstrapPeers)
	if err != nil {
		logger.Error("Error creating KAD", "err", err)
		log.Fatal(err)
	}

	go peerDiscoverer.Discover(ctx, kdht, nodeNamespace)

	songTable, err := song.SetupSongTable(ctx, h, logger)
	if err != nil {
		logger.Error("Setup global palylist error", "err", err)
		log.Fatal(err)
	}
	songTable.RegisterSongTableHandlers(ctx, h)

	////////////////////
	//.....TESTING......
	if len(bootstrapPeers) == 0 {
		select {}
	}

	songFile, err := song.NewSong("/Users/nikita/flow /p2p_music/.data/test/chsv.mp3")
	if err != nil {
		log.Fatal(err)
	}

	domainManager := song.NewSongManager(h, songTable, kdht, configs, logger)
	err = domainManager.PromoteSong(ctx, songFile)
	if err != nil {
		log.Fatal(err)
	}
	domainManager.RegisterSongStreamingProtocols(ctx)

	songProviders, err := domainManager.FindSongProviders(ctx, songFile)
	if err != nil {
		log.Fatal(err)
	}

	if len(songProviders) != 0 {
		err = domainManager.ReceiveSongStream(ctx, songFile, songProviders[len(songProviders)-1].ID)
		if err != nil {
			log.Fatal(err)
		}
	}
	// //////////////////
	// .....TESTING......
}
