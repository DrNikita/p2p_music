package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"p2p-music/config"
	"p2p-music/internal/peerdiscovery"

	"github.com/multiformats/go-multiaddr"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	configs, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := peerdiscovery.SetupHost()

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

	peerdiscovery.Bootstrap(ctx, h, discoveryPeers, configs, logger)

	select {}
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
