package tests

import (
	"context"
	"fmt"
	"testing"

	cmn "mnwarm/internal/shared"

	log "github.com/ipfs/go-log/v2"
	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	relay "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

type RelayNode struct {
	Host host.Host
}

func StartRelayNode(ctx context.Context) (*RelayNode, error) {
	log.SetAllLoggers(log.LevelInfo)
	keyIndex := 3
	nodeOpt := cmn.GetLibp2pIdentity(keyIndex)

	resourceManager, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits))
	if err != nil {
		return nil, fmt.Errorf("could not create new resource manager: %v", err)
	}

	host, err := libp2p.New(
		nodeOpt,
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/1234"),
		libp2p.EnableRelay(),
		libp2p.EnableRelayService(relay.WithInfiniteLimits()),
		libp2p.ResourceManager(resourceManager),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create relay host: %v", err)
	}

	_, err = relay.New(host)
	if err != nil {
		return nil, fmt.Errorf("failed to start relay service: %v", err)
	}

	return &RelayNode{Host: host}, nil
}

func (r *RelayNode) AddrInfo() *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    r.Host.ID(),
		Addrs: r.Host.Addrs(),
	}
}

type NodeRunner struct {
	Host host.Host
}

func StartNodeRunner(ctx context.Context, relayAddrInfo *peer.AddrInfo) (*NodeRunner, error) {
	log.SetAllLoggers(log.LevelInfo)
	keyIndex := 7
	nodeOpt := cmn.GetLibp2pIdentity(keyIndex)

	host, err := libp2p.New(
		nodeOpt,
		libp2p.NoListenAddrs,
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node runner host: %v", err)
	}

	// Connect to the relay node
	err = host.Connect(ctx, *relayAddrInfo)
	if err != nil {
		return nil, fmt.Errorf("node runner failed to connect to relay: %v", err)
	}

	// Reserve a relay slot
	err = cmn.ReserveRelay(ctx, host, relayAddrInfo)
	if err != nil {
		return nil, fmt.Errorf("node runner failed to reserve relay: %v", err)
	}

	// Set up stream handler for custom protocol
	host.SetStreamHandler(protocol.ID("/customprotocol/1.0.0"), func(s network.Stream) {
		defer s.Close()
		buf := make([]byte, 1024)
		n, err := s.Read(buf)
		if err != nil {
			fmt.Errorf("Node Runner failed to read from stream: %v", err)
			return
		}
		message := string(buf[:n])
		fmt.Printf("Node Runner received: %s", message)
		response := "Hello from Node Runner"
		_, err = s.Write([]byte(response))
		if err != nil {
			fmt.Errorf("Node Runner failed to write response: %v", err)
			return
		}
	})

	return &NodeRunner{Host: host}, nil
}

func (n *NodeRunner) AddrInfo() *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    n.Host.ID(),
		Addrs: n.Host.Addrs(),
	}
}

type MobileClient struct {
	Host host.Host
}

func StartMobileClient(ctx context.Context) (*MobileClient, error) {
	log.SetAllLoggers(log.LevelInfo)
	keyIndex := 8
	nodeOpt := cmn.GetLibp2pIdentity(keyIndex)

	host, err := libp2p.New(
		nodeOpt,
		libp2p.NoListenAddrs,
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mobile client host: %w", err)
	}

	return &MobileClient{Host: host}, nil
}

type BootstrapServer struct {
	Host host.Host
}

// func StartBootstrapServer(ctx context.Context, keyIndex int, msgChan chan<- string) (*BootstrapServer, error) {
// 	log.SetAllLoggers(log.LevelInfo)
// 	nodeOpt := cmn.GetLibp2pIdentity(keyIndex)

// 	host, err := libp2p.New(
// 		nodeOpt,
// 		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"), // Random available port
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create bootstrap server host: %w", err)
// 	}
// 	kademliaDHT, err := dht.New(ctx, host, dht.Mode(dht.ModeServer))
// 	if err != nil {
// 		fmt.Errorf("DHT failed %v", err)
// 	} else {
// 		fmt.Print(kademliaDHT)
// 	}

// 	return &BootstrapServer{Host: host}, nil
// }

// func (b *BootstrapServer) AddrInfo() *peer.AddrInfo {
// 	return &peer.AddrInfo{
// 		ID:    b.Host.ID(),
// 		Addrs: b.Host.Addrs(),
// 	}
// }

func (mc *MobileClient) InitialConnection(ctx context.Context, relayAddrInfo *peer.AddrInfo, nodeRunnerAddrInfo *peer.AddrInfo) error {
	err := cmn.ConnectToRelay(ctx, mc.Host, relayAddrInfo)
	if err != nil {
		return fmt.Errorf("mobile client failed to connect to relay: %w", err)
	}

	relayAddr, err := cmn.AssembleRelay(*relayAddrInfo, *nodeRunnerAddrInfo)
	if err != nil {
		return fmt.Errorf("failed to assemble relay address: %w", err)
	}

	mc.Host.Peerstore().AddAddrs(relayAddr.ID, relayAddr.Addrs, peerstore.PermanentAddrTTL)

	err = mc.Host.Connect(ctx, relayAddr)
	if err != nil {
		return fmt.Errorf("mobile client failed to connect to node runner via relay: %w", err)
	}

	return nil
}

func (mc *MobileClient) CommunicateWithNodeRunner(ctx context.Context, nodeRunnerAddrInfo *peer.AddrInfo) error {
	s, err := mc.Host.NewStream(network.WithAllowLimitedConn(context.Background(), "customprotocol"), nodeRunnerAddrInfo.ID, protocol.ID("/customprotocol/1.0.0"))
	if err != nil {
		return fmt.Errorf("failed to create stream to node runner: %w", err)
	}
	defer s.Close()

	message := "Hello from mobile client"
	_, err = s.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("mobile client failed to write to stream: %w", err)
	}
	fmt.Printf("Mobile client sent: %s", message)

	// Read the response
	buf := make([]byte, 1024)
	n, err := s.Read(buf)
	if err != nil {
		return fmt.Errorf("mobile client failed to read from stream: %w", err)
	}
	response := string(buf[:n])
	fmt.Printf("Mobile client received: %s", response)

	// Validate the response
	if response != "Hello from Node Runner" {
		return fmt.Errorf("unexpected response from node runner: %s", response)
	}

	return nil
}

func (mc *MobileClient) CommunicateWithNodeRunnerOld(t *testing.T, relayAddrInfo *peer.AddrInfo, nodeRunnerAddrInfo *peer.AddrInfo) {
	targetRelayedInfo, err := cmn.AssembleRelay(*relayAddrInfo, *nodeRunnerAddrInfo)
	if err != nil {
		t.Errorf("error in assembleRelay for peer %s: %+v", nodeRunnerAddrInfo, err)
	}

	if err := mc.Host.Connect(context.Background(), targetRelayedInfo); err != nil {
		t.Errorf("Connection failed via relay: %v", err)
	}
	// add a timeout?
	t.Logf("we have connected to peer %s via relay %s", nodeRunnerAddrInfo.ID, relayAddrInfo.ID)

	t.Logf("peerswithaddrs: %v", mc.Host.Peerstore().PeersWithAddrs())
	t.Helper()
	t.Logf("Connected level of %s is %+v", nodeRunnerAddrInfo.ID, mc.Host.Network().Connectedness(nodeRunnerAddrInfo.ID))

	s, err := mc.Host.NewStream(network.WithAllowLimitedConn(context.Background(), "customprotocol"), nodeRunnerAddrInfo.ID, protocol.ID("/customprotocol/1.0.0"))
	if err != nil {
		t.Fatalf("Failed to create stream to node runner: %v", err)
	} else {
		t.Log("stream started")
	}
	defer s.Close()
	message := "Hello from mobile client"
	_, err = s.Write([]byte(message))
	if err != nil {
		t.Fatalf("Mobile client failed to write to stream: %v", err)
	}
	t.Logf("Mobile client sent: %s", message)
	buf := make([]byte, 1024)
	n, err := s.Read(buf)
	if err != nil {
		t.Fatalf("Mobile client failed to read from stream: %v", err)
	}
	response := string(buf[:n])
	t.Logf("Mobile client received: %s", response)
}
