package peerdiscovery

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"p2p-music/config"
	"p2p-music/internal/db"
	"p2p-music/internal/song"
	"time"

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
	fmt.Println("0--------------------------")
	// Peer discovery
	peerDiscoverer := NewDHTManager(h, logger)
	kdht, err := peerDiscoverer.NewDHT(ctx, bootstrapPeers)
	if err != nil {
		logger.Error("Error creating KAD", "err", err)
		log.Fatal(err)
	}

	go peerDiscoverer.Discover(ctx, kdht, nodeNamespace)

	//w8 for Discovery start
	time.Sleep(3 * time.Second)

	fmt.Println("1--------------------------")
	// Global song list initialization
	songTable, err := song.SetupSongTable(ctx, h, logger)
	if err != nil {
		logger.Error("Setup global palylist error", "err", err)
		log.Fatal(err)
	}
	songTable.RegisterSongTableHandlers(ctx, h)

	fmt.Println("2--------------------------")
	store, closeDBConn, err := db.InitDB(logger)
	if err != nil {
		logger.Error("Failed to init BoltDB", "err", err)
		log.Fatal(err)
	}

	fmt.Println("3--------------------------")

	songTableManager := song.NewSongManager(h, songTable, kdht, store, configs, logger)
	songTableManager.RegisterSongStreamingProtocols(ctx)

	fmt.Println("4--------------------------")

	////////////////////
	//.....TESTING......
	if len(bootstrapPeers) == 0 {
		select {}
	}
	fmt.Println("5--------------------------")

	songFilePath := "/home/nikita/workspace/p2p_music/.data/test/Bruno.mp3"
	song, err := song.NewSong(songFilePath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("6--------------------------")

	err = songTableManager.PromoteSong(ctx, song, songFilePath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("7--------------------------")

	songProviders, err := songTableManager.FindSongProviders(ctx, song)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("8--------------------------")

	if len(songProviders) < 1 {
		fmt.Println("___NO_PROVIDERS___")
		return nil
	}
	fmt.Println("9--------------------------")

	_, err = songTableManager.ReceiveSongStream(ctx, song, songProviders[len(songProviders)-1].ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("10--------------------------")
	// store.SaveFilePath(ctx, )

	// //////////////////
	// .....TESTING......

	return closeDBConn
}
