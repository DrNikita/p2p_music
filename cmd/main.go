package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"p2p-music/config"
	"p2p-music/discovery"
	"p2p-music/domain"
	"p2p-music/domain/store"
	"time"

	"github.com/multiformats/go-multiaddr"

	_ "github.com/joho/godotenv/autoload"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

const (
	defaultAddr          = "/ip4/0.0.0.0/tcp/0"
	nodeNamespace string = "music"
)

func main() {
	// Start with the default scaling limits.
	scalingLimits := rcmgr.DefaultLimits

	// Add limits around included libp2p protocols
	libp2p.SetDefaultServiceLimits(&scalingLimits)

	// Turn the scaling limits into a concrete set of limits using `.AutoScale`. This
	// scales the limits proportional to your system memory.
	scaledDefaultLimits := scalingLimits.AutoScale()

	// Tweak certain settings
	cfg := rcmgr.PartialLimitConfig{
		System: rcmgr.ResourceLimits{
			// Allow unlimited outbound streams
			StreamsOutbound: rcmgr.Unlimited,
		},
		// Everything else is default. The exact values will come from `scaledDefaultLimits` above.
	}

	// Create our limits by using our cfg and replacing the default values with values from `scaledDefaultLimits`
	limits := cfg.Build(scaledDefaultLimits)

	// The resource manager expects a limiter, se we create one from our limits.
	limiter := rcmgr.NewFixedLimiter(limits)

	// Metrics are enabled by default. If you want to disable metrics, use the
	// WithMetricsDisabled option
	// Initialize the resource manager
	rm, err := rcmgr.NewResourceManager(limiter, rcmgr.WithMetricsDisabled())
	if err != nil {
		panic(err)
	}

	_, err = config.MustConfig()
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(defaultAddr),
		libp2p.ResourceManager(rm),
		libp2p.EnableAutoNATv2(),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Node addresses:")
	for _, addr := range h.Addrs() {
		fmt.Printf("%s/p2p/%s\n", addr, h.ID())
	}

	discoveryPeers := []multiaddr.Multiaddr{}

	cmdDiscoveryPeers := getCmdPeerDiscovery()
	for _, cmdPeer := range cmdDiscoveryPeers {
		peerMultiaddr, err := multiaddr.NewMultiaddr(cmdPeer)
		if err != nil {
			log.Fatal(err)
		}
		discoveryPeers = append(discoveryPeers, peerMultiaddr)
		fmt.Println("discovery PEER:", cmdPeer)
	}

	// create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Fatal(err)
	}

	kdht, err := discovery.NewDHT(ctx, h, discoveryPeers, logger)
	if err != nil {
		log.Fatal(err)
	}

	go discovery.Discover(ctx, h, kdht, nodeNamespace, logger)

	time.Sleep(5 * time.Second)

	gp, err := store.SetupGlobalPlaylist(ctx, ps, h, logger)
	if err != nil {
		log.Fatal(err)
	}
	gp.RegisterGetPlaylistHandler(ctx, h)

	////////////////
	/*
	   TESTING
	*/
	///////////////

	if len(discoveryPeers) == 0 {
		select {}
	}

	song, err := store.NewSong("/Users/nikita/flow /p2p_music/.data/music/chsv.mp3")
	if err != nil {
		log.Fatal(err)
	}

	domainManager := domain.NewDomainManager(h, gp, kdht, logger)
	err = domainManager.PromoteSong(ctx, song)
	if err != nil {
		log.Fatal(err)
	}
	domainManager.RegisterProtocols(ctx)

	songProviders, err := domainManager.FindSongProviders(ctx, song)
	if err != nil {
		log.Fatal(err)
	}

	if len(songProviders) != 0 {
		err = domainManager.ReceiveSongStream(ctx, song, songProviders[len(songProviders)-1].ID)
		if err != nil {
			log.Fatal(err)
		}
	}

	select {}

	////////////////
	/*
	   TESTING
	*/
	///////////////
}

func getCmdPeerDiscovery() []string {
	peerDiscovery := make([]string, 0)
	var found bool

	cmdArgs := os.Args
	for _, arg := range cmdArgs {
		if found && (peerAddrPrefix(arg) == "/ip4/" || peerAddrPrefix(arg) == "/ip6/") {
			peerDiscovery = append(peerDiscovery, arg)
			break
		}
		if arg == "-discovery" {
			found = true
		}
		fmt.Println(arg)
	}

	return peerDiscovery
}

func peerAddrPrefix(peerAddr string) string {
	runePeerAddr := []rune(peerAddr)
	if len(runePeerAddr) < 6 {
		return ""
	}
	return string(runePeerAddr[:5])
}
