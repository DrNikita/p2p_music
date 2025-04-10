package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"p2p-music/config"
	"p2p-music/internal/peerdiscovery"
	"p2p-music/tui/model"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

	closeDB, songTable := peerdiscovery.Bootstrap(ctx, h, discoveryPeers, configs, logger)

	time.Sleep(time.Second)

	p := tea.NewProgram(model.InitTea(songTable))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	defer closeDB()
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
