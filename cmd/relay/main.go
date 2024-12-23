// relay.go
package main

import (
	"context"
	"flag"
	"fmt" // Added import for io
	"io"
	"strings"
	"time"

	// tptu "github.com/libp2p/go-libp2p/p2p/net/upgrader"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"

	logging "github.com/ipfs/go-log/v2"
	libp2p "github.com/libp2p/go-libp2p"

	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"

	cmn "mnwarm/internal/shared"

	multiaddr "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("relaylog")

var relayerPrivateKeys = []string{
	//boots
	"CAESQAA7xVQKsQ5VAC5ge+XsixR7YnDkzuHa4nrY8xWXGK3fo9yN1Eaiat9Vn1iwaVQDqTjywVP303ojVLxXcQ9ze4E=",
	// pid: 12D3KooWLr1gYejUTeriAsSu6roR2aQ423G3Q4fFTqzqSwTsMz9n
	"CAESQMCYbjRpXBDUnIpDyqY+mA3n7z9gF3CaggWTknd90LauHUcz8ldNtlUchFATmMSE1r/NMnSpEBbLvzWQKq3N45s=",
	// pid: 12D3KooWBnext3VBZZuBwGn3YahAZjf49oqYckfx64VpzH6dyU1p
	"CAESQB1Y1Li0Wd4KcvMvbv5/+CTG79axzl3R8yTuzWOckMgmNAzZqxim5E/7e9mgd87FTMPQNHqiItqTFwHJeMxr0H8=",
	// pid: 12D3KooWDKYjXDDgSGzhEYWYtDvfP9pMtGNY1vnAwRsSp2CwCWHL

	//relays
	"CAESQHMEeM3iNIIxNThxIfnuO5FJ0oUQJy8V7TFD80lGziBE7SuPw2wckCrFRihVDaw0e6PkDCwsh/6u3UgBxB3OTFo=",
	//12D3KooWRnBKUEkAEpsoCoEiuhxKBJ5j2Bdop6PGxFMvd4PwoevM
	"CAESQP3Pu7TVp2RSVIZykj65/MDXm/eiTOfLGH3xCWQVmUoC67MkFWUEOd6QERl1Y4Xvi1Rt+d36UuaFXanT+hVUDAY=",
	//12D3KooWRgSQnguL2DYkXUXqCLiRQ35PEX4eEH3havy2X18AVALd
	"CAESQDE2IToG5mWwzWEeXt3/OVbx9XyE743DTenPFUG8M06IQXSarkNhuxNEJisnWeuDvaoaM/fNJNMqhPR81NL3Pio=",
	//12D3KooWEDso33ti9KsKmD2g2egNmw6BXgch7V5vFz1TziuNYybo

	//nodes
	"CAESQFffsVM3eUXLozmXkBM2FSSVhEmo/Cq5RlXOAAaniTdCu3EQ6Zf7lQDasCj6IXyTihFQWZB+nmGFn/ZAA5y5egk=",
	//12D3KooWNS4QQxwNURwoYoXmGjH9AQkagcGTjRUQT33P4i4FKQsi
	"CAESQCSHrfyzNZkxwoNmXI1wx5Lvr6o4+kGxGepFH0AfYlKthyON+1hQRjLJQaBAQLrr1cfMHFFoC40X62DQIhL246U=",
	//12D3KooWJuteouY1d5SYFcAUAYDVPjFD8MUBgqsdjZfBkAecCS2Y
	"CAESQDyiSqC9Jez8wKSQs74YJalAegamjVKHbnaN35pfe6Gk21WVgCzfvBdLVoRj8XXny/k1LtSOhPZWNz0rWKCOYpk=",
	//12D3KooWQaZ9Ppi8A2hcEspJhewfPqKjtXu4vx7FQPaUGnHXWpNL
}

// var NodeRunnerProtocol = protocol.ID("/customprotocol/request-node-runner/1.0.0")
// var NodeRunnerProtocol = protocol.ID(identify.ID)
var NodeRunnerProtocol = protocol.ID("/customprotocol/1.0.0")

func RelayIdentity(keyIndex int) (libp2p.Option, error) {
	if keyIndex < 0 || keyIndex >= len(relayerPrivateKeys) {
		return nil, fmt.Errorf("invalid key index: %d", keyIndex)
	}

	keyStr := relayerPrivateKeys[keyIndex]
	keyBytes, err := crypto.ConfigDecodeKey(keyStr)
	if err != nil {
		return nil, fmt.Errorf("decode private key failed: %w", err)
	}

	privKey, err := crypto.UnmarshalPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshal key failed: %w", err)
	}

	return libp2p.Identity(privKey), nil
}

func initializeLogger() {

	// logging.SetAllLoggers(logging.LevelWarn)
	logging.SetAllLoggers(logging.LevelInfo)

	// logging.SetLogLevel("dht", "error") // get rid of  network size estimator track peers: expected bucket size number of peers

	logging.SetLogLevel("relaylog", "debug")
}

func parseCommandLineArgs() (int, string, int, string) {
	listenPort := flag.Int("port", 1237, "TCP port to listen on")
	bootstrapPeers := flag.String("bootstrap", "", "Comma separated bootstrap peer multiaddrs")
	keyIndex := flag.Int("key", 0, "Relayer private key index")
	noderunnerID := flag.String("noderunner", "", "Peer ID of the Node Runner")
	flag.Parse()

	return *listenPort, *bootstrapPeers, *keyIndex, *noderunnerID
}

func getRelayIdentity(keyIndex int) libp2p.Option {
	relayOpt, err := RelayIdentity(keyIndex)
	if err != nil {
		log.Fatalf("relay identity error: %v", err)
	}
	return relayOpt
}

func createHost(ctx context.Context, relayOpt libp2p.Option, listenPort int) host.Host {
	ListenAddrs := func(cfg *config.Config) error {
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
	}

	rcmgr, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits))
	if err != nil {
		log.Fatalf("could not create new resource manager: %w", err)
	}

	host, err := libp2p.New(
		relayOpt,
		ListenAddrs,
		libp2p.EnableRelay(),
		libp2p.EnableRelayService(relay.WithInfiniteLimits()), // do a config with limited istead?
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableHolePunching(),
		libp2p.ForceReachabilityPublic(),
		libp2p.Security(noise.ID, noise.New),
		// libp2p.Security(
		// 	noise.ID,
		// 	func(id protocol.ID, privkey crypto.PrivKey, muxers []tptu.StreamMuxer) (*noise.SessionTransport, error) {
		// 		tp, err := noise.New(id, privkey, muxers)
		// 		if err != nil {
		// 			return nil, err
		// 		}
		// 		return tp.WithSessionOptions(noise.Prologue(prologue))
		// 	},
		// ),
		libp2p.ResourceManager(rcmgr),
	)
	if err != nil {
		log.Fatal(err)
	}
	return host
}

func setupRelayService(host host.Host) (*relay.Relay, relay.MetricsTracer) {
	mt := relay.NewMetricsTracer()
	log.Debugf("Relay timeouts: %d %d %d",
		relay.ConnectTimeout,
		relay.StreamTimeout,
		relay.HandshakeTimeout)

	relayService, err := relay.New(host, relay.WithInfiniteLimits(), relay.WithMetricsTracer(mt))
	log.Debugf("relayservice %+v", relayService)
	// mt.RelayStatus(true)
	// limit resources?
	// mt.RelayStatus(true)
	// var status pb.Status
	if err != nil {
		log.Fatalf("Failed to instantiate the relay: %v", err)
		return nil, nil
	}
	return relayService, mt
}

func logHostInfo(host host.Host) {
	log.Infof("Relay node is running. Peer ID: %s", host.ID())
	log.Info("Listening on:")
	for _, addr := range host.Addrs() {
		log.Infof("%s/p2p/%s", addr, host.ID())
	}
}

func createDHT(ctx context.Context, host host.Host) *dht.IpfsDHT {
	kademliaDHT, err := dht.New(ctx, host, dht.Mode(dht.ModeServer))
	if err != nil {
		log.Fatal(err)
	}
	return kademliaDHT
}

func bootstrapDHT(ctx context.Context, kademliaDHT *dht.IpfsDHT) {
	log.Info("Bootstrapping DHT")
	if err := kademliaDHT.Bootstrap(ctx); err != nil {
		log.Fatal(err)
	}
}

func connectToBootstrapPeers(ctx context.Context, host host.Host, bootstrapPeers string) {
	if bootstrapPeers != "" {
		peerAddrs := strings.Split(bootstrapPeers, ",")
		for _, addr := range peerAddrs {
			addr = strings.TrimSpace(addr)
			if addr == "" {
				log.Warn("empty bootstrap addr")
				continue
			}

			maddr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				log.Errorf("invalid bootstrap addr '%s': %v", addr, err)
				continue
			}

			peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
			if err != nil {
				log.Errorf("get peer info failed for '%s': %v", addr, err)
				continue
			}

			host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)

			if err := host.Connect(ctx, *peerInfo); err != nil {
				log.Errorf("connect to bootstrap peer %s failed: %v", peerInfo.ID, err)
				continue
			}
			log.Infof("Connected to bootstrap peer %s", peerInfo.ID)
		}
	}
}

func setupDHTRefresh(kademliaDHT *dht.IpfsDHT) {
	go func() {
		for {
			time.Sleep(60 * time.Second)
			kademliaDHT.RefreshRoutingTable()
			peers := kademliaDHT.RoutingTable().ListPeers()
			log.Infof("Routing table peers (%d): %v", len(peers), peers)
			// log.Infof("Relay Status (%d): %v", mt.RelayStatus())
			// var status pb.Status
			// mt.ConnectionRequestHandled(status)
			// log.Info(status)
			// mt.ReservationRequestHandled(status)
			// log.Info(status)
		}
	}()
}

func handleStream(stream network.Stream) {
	log.Infof("%s: Received stream status request from %s. Node guid: %s", stream.Conn().LocalPeer(), stream.Conn().RemotePeer())
	log.Error("NEW STREAM!!!!!")
	peerID := stream.Conn().RemotePeer()
	log.Infof("relay got new stream from %s", peerID)
	log.Infof("direction, opened, limited: %v", stream.Stat())

	defer stream.Close()

	// 12D3KooWJuteouY1d5SYFcAUAYDVPjFD8MUBgqsdjZfBkAecCS2Y
	// Return a string to the mobile client

	buf := make([]byte, 5)
	maxRetry := 5

	for i := 0; i < maxRetry; i++ {
		log.Infof("attempting to read from stream, attempt %d", i)
		n, err := stream.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Warnf("received EOF from %s, retrying read attempt %d/%d", peerID, i, maxRetry)
				time.Sleep(1 * time.Second)
				continue
			} else {
				log.Errorf("error reading from stream: %v", err)
				return
			}
		}

		received := string(buf[:n])
		log.Infof("received message: %s from %s", received, peerID)

		if received == "PING\n" {
			log.Infof("received PING from %s, responding with PONG", peerID)
			if _, err = fmt.Fprintf(stream, "PONG\n"); err != nil {
				log.Errorf("error writing PONG to stream: %v", err)
				return
			}
			log.Infof("sent PONG to %s", peerID)
		} else {
			log.Warnf("unexpected message from %s: %s", peerID, received)
		}

		break
	}
}

func main() {
	initializeLogger()
	identify.ActivationThresh = 1

	cmn.ParseCmdArgs()
	relayAddrStr, keyIndexInt, bootstrapAddrs, err := cmn.ParseCmdArgs()
	log.Infof("%v, %v, %v ", relayAddrStr, keyIndexInt, bootstrapAddrs)
	listenPort := 1240
	// listenPort, bootstrapPeers, keyIndex, noderunnerIDStr := parseCommandLineArgs()
	log.Infof("%v %v %v %v", listenPort, bootstrapAddrs, keyIndexInt)

	nodeOpt, err := cmn.GetLibp2pIdentity(keyIndexInt)

	if err != nil {
		log.Errorf("error in startup %v", err)
	}
	ctx := context.Background()

	host := createHost(ctx, nodeOpt, listenPort)

	relayService, metrics := setupRelayService(host)

	log.Info(relayService, metrics)
	logHostInfo(host)

	kademliaDHT := createDHT(ctx, host)

	bootstrapDHT(ctx, kademliaDHT)

	bootstrapPeers, err := cmn.ParseBootstrap(bootstrapAddrs)
	if len(bootstrapPeers) == 0 {
		log.Fatal("no valid bootstrap addrs")
	}

	cmn.ConnectToBootstrapPeers(ctx, host, bootstrapPeers)
	// connectToBootstrapPeers(ctx, host, bootstrapPeers)

	time.Sleep(5 * time.Second)

	logHostInfo(host)

	setupDHTRefresh(kademliaDHT)

	// host.SetStreamHandler(NodeRunnerProtocol, func(s network.Stream) {
	// 	handleStream(strea)
	// })
	host.SetStreamHandler(protocol.ID(NodeRunnerProtocol), handleStream)

	log.Info("Relay is ready to handle Node Runner ID requests")

	select {}
}
