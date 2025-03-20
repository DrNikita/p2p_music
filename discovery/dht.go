package discovery

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

const (
	nodeNamespace string = "music"
	playlistTopic string = "playlist"
)

type DiscoveryService struct {
	h      host.Host
	kdht   *dht.IpfsDHT
	logger *slog.Logger
	// mb can add PlaylistStream here
}

func NewDiscoverService(ctx context.Context, h host.Host, bootstrapPeers []multiaddr.Multiaddr, logger *slog.Logger) (DiscoveryService, error) {
	kdht, err := newDHT(ctx, h, bootstrapPeers, logger)
	if err != nil {
		return DiscoveryService{}, err
	}

	time.Sleep(time.Second)

	go discover(ctx, h, kdht, nodeNamespace, logger)

	return DiscoveryService{
		h:      h,
		kdht:   kdht,
		logger: logger,
	}, nil
}

func newDHT(ctx context.Context, h host.Host, bootstrapPeers []multiaddr.Multiaddr, logger *slog.Logger) (*dht.IpfsDHT, error) {
	var opts []dht.Option

	// if no bootstrap peers give this peer act as a bootstraping node
	// other peers can use this peers ipfs address for peer discovery via dht
	if len(bootstrapPeers) == 0 {
		opts = append(opts, dht.Mode(dht.ModeServer))
	}

	kdht, err := dht.New(ctx, h, opts...)
	if err != nil {
		logger.Error("Failed to create new DHT", "err", err)
		return nil, err
	}

	if err := kdht.Bootstrap(ctx); err != nil {
		logger.Error("Failed to Bootstrap", "err", err)
		return nil, err
	}

	if len(bootstrapPeers) == 0 {
		return kdht, nil
	}

	errChan := make(chan error, len(bootstrapPeers))
	var wg sync.WaitGroup
	for _, peerAddr := range bootstrapPeers {
		peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			logger.Error("Error", "err", err)
			return nil, err
		}

		wg.Add(1)
		go func(pi *peer.AddrInfo) {
			defer wg.Done()
			if err := h.Connect(ctx, *peerInfo); err != nil {
				logger.Error("Error while connecting to node", "PeerID", peerInfo.ID, "err", err)
				errChan <- err
				return
			}
			logger.Info("Connection established with bootstrap node", "PeerID", peerInfo.ID)
			errChan <- nil
		}(peerInfo)
	}
	wg.Wait()
	close(errChan)

	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("context timeout")
		case err, ok := <-errChan:
			if err == nil && ok {
				return kdht, nil
			}
		default:
			return nil, errors.New("couln't connect to any specified bootstrap nodes")
		}
	}
}

func discover(ctx context.Context, h host.Host, kdht *dht.IpfsDHT, rendezvous string, logger *slog.Logger) {
	routingDiscovery := drouting.NewRoutingDiscovery(kdht)
	logger.Info("Advertising rendezvous point", "rendezvous", rendezvous)
	if _, err := routingDiscovery.Advertise(ctx, rendezvous); err != nil {
		logger.Error("Failed to advertise rendezvous point", "err", err)
		return
	}

	// Periodically find peers and connect to them
	ticker := time.NewTicker(10 * time.Second) // Check for peers every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peers, err := routingDiscovery.FindPeers(ctx, rendezvous)
			if err != nil {
				logger.Error("Failed to find peers", "err", err)
				continue
			}

			for peer := range peers {
				if peer.ID == h.ID() {
					continue // Skip self
				}

				if h.Network().Connectedness(peer.ID) != network.Connected {
					_, err := h.Network().DialPeer(ctx, peer.ID)
					if err != nil {
						logger.Error("Failed to connect to peer", "PeerID", peer.ID, "err", err)
						continue
					}
					logger.Info("Connected to peer", "PeerID", peer.ID)
				}
			}
		}
	}
}

func (ds DiscoveryService) ProvideSong(ctx context.Context, filePath string) {
	contentID, err := generateCIDFromFile(filePath)
	if err := ds.kdht.Provide(ctx, contentID, true); err != nil {
		ds.logger.Error("Failed to provide content", "err", err)
		return
	}

	providers, err := ds.kdht.FindProviders(ctx, contentID)
	if err != nil {
		ds.logger.Error("Failed to find providers", "err", err)
		return
	}
	ds.logger.Info("Providers found", "count", len(providers), "providers", providers)
}

func generateCIDFromFile(filePath string) (cid.Cid, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create a SHA-256 hasher
	hasher := sha256.New()

	// Stream the file content through the hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return cid.Undef, fmt.Errorf("failed to hash file: %w", err)
	}

	// Encode the hash as a multihash
	mh, err := multihash.Encode(hasher.Sum(nil), multihash.SHA2_256)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to encode multihash: %w", err)
	}

	// Create a CID using the multihash
	return cid.NewCidV1(cid.Raw, mh), nil
}
