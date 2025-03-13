package discovery

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
)

func NewDHT(ctx context.Context, h host.Host, bootstrapPeers []multiaddr.Multiaddr, logger *slog.Logger) (*dht.IpfsDHT, error) {
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
			} else {
				logger.Info("Connection established with bootstrap node", "PeerID", peerInfo.ID)
				errChan <- nil
			}
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

func Discover(ctx context.Context, h host.Host, kdht *dht.IpfsDHT, rendezvous string, logger *slog.Logger) {
	// Waiting for dht bootstrapping finish
	time.Sleep(1 * time.Second)

	routingDiscovery := drouting.NewRoutingDiscovery(kdht)

	peers := kdht.RoutingTable().ListPeers()
	logger.Info("Existing peers", "count", len(peers))

	var advertiseRetries int

	if kdht.Mode() != dht.ModeServer {
		advertiseRetries = 3
	}

	for attempt := range advertiseRetries {
		_, err := routingDiscovery.Advertise(ctx, rendezvous)
		if err == nil {
			break
		}
		logger.Error("Failed to advertise, retrying...", "err", err, "attempt", attempt+1)
		time.Sleep(time.Second * time.Duration(attempt+1))
	}

	ticker := time.NewTicker(time.Millisecond * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peers, err := routingDiscovery.FindPeers(ctx, rendezvous)
			if err != nil {
				logger.Error("Failed to find peers", "err", err)
			}

			for peer := range peers {
				if peer.ID == h.ID() {
					continue
				}

				if h.Network().Connectedness(peer.ID) != network.Connected {
					_, err := h.Network().DialPeer(ctx, peer.ID)
					if err != nil {
						continue
					}
					logger.Info("Connected to peer", "PeerID", peer.ID)
				}
			}
		}
	}
}
