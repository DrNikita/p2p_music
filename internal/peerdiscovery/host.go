package peerdiscovery

import (
	"fmt"
	"log"

	_ "github.com/joho/godotenv/autoload"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/multiformats/go-multiaddr"
)

const (
	defaultAddr = "/ip4/0.0.0.0/tcp/0"
)

func SetupHost() host.Host {
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

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(defaultAddr),
		libp2p.ResourceManager(rm),
		libp2p.EnableAutoNATv2(),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Available addresses:")
	for _, addr := range h.Addrs() {
		fullAddr := addr.Encapsulate(multiaddr.StringCast(fmt.Sprintf("/p2p/%s", h.ID())))
		fmt.Println(fullAddr)
	}

	return h
}
