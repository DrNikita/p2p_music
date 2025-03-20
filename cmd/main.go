package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"p2p_transfer/config"
	"p2p_transfer/discovery"

	"github.com/multiformats/go-multiaddr"

	_ "github.com/joho/godotenv/autoload"
	"github.com/libp2p/go-libp2p"
)

const defaultAddr = "/ip4/0.0.0.0/tcp/0"

func main() {
	_, err := config.MustConfig()
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
		libp2p.EnableAutoNATv2(),
	)
	if err != nil {
		panic(err)
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

	discoveryService, err := discovery.NewDiscoverService(ctx, h, discoveryPeers, logger)
	if err != nil {
		panic(err)
	}

	discoveryService.ProvideSong(ctx, "/Users/nikita/flow /p2p_music/.data/music/pirat.mp3")

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
