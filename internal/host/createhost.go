package create

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	relay "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("bootlog")

// Returns:
// - host.Host: libp2p host.
// - *dht.IpfsDHT:  DHT instance
// - error: any errors during creation
func CreateHost(
	ctx context.Context,
	nodeType NodeType,
	relayInfo *peer.AddrInfo,
	listenPort int,
) (host.Host, *dht.IpfsDHT, error) {
	var kademliaDHT *dht.IpfsDHT
	mt := autorelay.NewMetricsTracer()

	resourceManager, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits))
	if err != nil {
		return nil, nil, fmt.Errorf("could not create new resource manager: %w", err)
	}

	opts := []libp2p.Option{
		libp2p.ResourceManager(resourceManager),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableHolePunching(),
	}

	switch nodeType {
	case Bootstrap:
		opts = append(opts,
			libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
			libp2p.ForceReachabilityPublic(),
		)

	case Relay:
		opts = append(opts, func(cfg *config.Config) error {
			addrs := []string{
				fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),
				"/ip6/::/tcp/0",
			}
			listenAddrs := make([]multiaddr.Multiaddr, 0, len(addrs))
			for _, s := range addrs {
				addr, err := multiaddr.NewMultiaddr(s)
				if err != nil {
					return err
				}
				listenAddrs = append(listenAddrs, addr)
			}
			return cfg.Apply(libp2p.ListenAddrs(listenAddrs...))
		})

		opts = append(opts,
			libp2p.EnableRelay(),
			libp2p.EnableRelayService(relay.WithInfiniteLimits()),
			libp2p.ForceReachabilityPublic(),
		)

	case MobileClient, NodeRunner:
		opts = append(opts, func(cfg *config.Config) error {
			addrs := []string{
				"/ip4/0.0.0.0/tcp/0",
				"/ip6/::/tcp/0",
				"/ip4/0.0.0.0/udp/0/quic",
				"/ip6/::/udp/0/quic",
			}
			listenAddrs := make([]multiaddr.Multiaddr, 0, len(addrs))
			for _, s := range addrs {
				addr, err := multiaddr.NewMultiaddr(s)
				if err != nil {
					return err
				}
				listenAddrs = append(listenAddrs, addr)
			}
			return cfg.Apply(libp2p.ListenAddrs(listenAddrs...))
		})

		opts = append(opts, libp2p.EnableRelay())
		if relayInfo != nil {
			opts = append(opts, libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{*relayInfo}, autorelay.WithMetricsTracer(mt)))
		}

		opts = append(opts,
			libp2p.ForceReachabilityPrivate(),
			libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
				var err error
				kademliaDHT, err = dht.New(ctx, h, dht.Mode(dht.ModeClient))
				if err != nil {
					return nil, fmt.Errorf("failed to create DHT: %w", err)
				}
				return kademliaDHT, nil
			}),
		)
	default:
		return nil, nil, fmt.Errorf("unknown NodeType: %d", nodeType)
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Libp2p host: %w", err)
	}

	log.Infof("Host created with ID: %s", h.ID())

	return h, kademliaDHT, nil
}
