package tests

import (
	"context"
	"fmt"

	cmn "mnwarm/internal/shared"

	log "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	relay "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

const RelayProtocol = "/relay/reservation/1.0.0"

type Relay struct {
	Host host.Host
}

func StartRelay(ctx context.Context) (*Relay, error) {
	log.SetAllLoggers(log.LevelInfo)
	keyIndex := 3

	nodeOpt, err := cmn.GetLibp2pIdentity(keyIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	resourceManager, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits))
	if err != nil {
		return nil, fmt.Errorf("resource manager error: %w", err)
	}

	h, err := libp2p.New(
		nodeOpt,
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/1234"),
		libp2p.EnableRelay(),
		libp2p.EnableRelayService(relay.WithInfiniteLimits()),
		libp2p.ResourceManager(resourceManager),
	)
	if err != nil {
		return nil, fmt.Errorf("relay host creation failed: %w", err)
	}

	_, err = relay.New(h)
	if err != nil {
		_ = h.Close()
		return nil, fmt.Errorf("relay service failed: %w", err)
	}

	h.SetStreamHandler(protocol.ID(RelayProtocol), func(s network.Stream) {
		defer s.Close()
		buf := make([]byte, 1024)
		n, err := s.Read(buf)
		if err != nil {
			fmt.Errorf("Read error: %v", err)
			return
		}
		request := string(buf[:n])
		fmt.Printf("Received reservation: %s", request)

		response := "Reservation successful"
		if _, err := s.Write([]byte(response)); err != nil {
			fmt.Errorf("Write error: %v", err)
		}
	})

	return &Relay{Host: h}, nil
}

func (r *Relay) AddrInfo() *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    r.Host.ID(),
		Addrs: r.Host.Addrs(),
	}
}

type Runner struct {
	Host host.Host
}

func StartRunner(ctx context.Context, relayInfo *peer.AddrInfo) (*Runner, error) {
	log.SetAllLoggers(log.LevelInfo)
	keyIndex := 7

	nodeOpt, err := cmn.GetLibp2pIdentity(keyIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	h, err := libp2p.New(
		nodeOpt,
		libp2p.NoListenAddrs,
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		return nil, fmt.Errorf("runner host creation failed: %w", err)
	}

	h.Peerstore().AddAddrs(relayInfo.ID, relayInfo.Addrs, peerstore.PermanentAddrTTL)

	if err := h.Connect(ctx, *relayInfo); err != nil {
		_ = h.Close()
		return nil, fmt.Errorf("connect to relay failed: %w", err)
	}

	if err := cmn.ReserveRelay(ctx, h, relayInfo); err != nil {
		_ = h.Close()
		return nil, fmt.Errorf("reserve relay failed: %w", err)
	}

	h.SetStreamHandler(protocol.ID("/customprotocol/1.0.0"), func(s network.Stream) {
		defer s.Close()
		buf := make([]byte, 1024)
		n, err := s.Read(buf)
		if err != nil {
			fmt.Errorf("Read error: %v", err)
			return
		}
		message := string(buf[:n])
		fmt.Printf("Received: %s", message)
		response := "Hello from Runner"
		if _, err := s.Write([]byte(response)); err != nil {
			fmt.Errorf("Write error: %v", err)
		}
	})

	return &Runner{Host: h}, nil
}

func (r *Runner) AddrInfo() *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    r.Host.ID(),
		Addrs: r.Host.Addrs(),
	}
}

type Client struct {
	Host host.Host
}

func StartClient(ctx context.Context) (*Client, error) {
	log.SetAllLoggers(log.LevelInfo)
	keyIndex := 8
	nodeOpt, err := cmn.GetLibp2pIdentity(keyIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	h, err := libp2p.New(
		nodeOpt,
		libp2p.NoListenAddrs,
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		return nil, fmt.Errorf("client host creation failed: %w", err)
	}

	return &Client{Host: h}, nil
}

func (c *Client) Connect(ctx context.Context, relayInfo, runnerInfo *peer.AddrInfo) error {
	if err := cmn.ConnectToRelay(ctx, c.Host, relayInfo); err != nil {
		return fmt.Errorf("connect to relay failed: %w", err)
	}

	relayAddr, err := cmn.AssembleRelay(*relayInfo, *runnerInfo)
	if err != nil {
		return fmt.Errorf("assemble relay address failed: %w", err)
	}

	c.Host.Peerstore().AddAddrs(relayAddr.ID, relayAddr.Addrs, peerstore.PermanentAddrTTL)

	if err := c.Host.Connect(ctx, relayAddr); err != nil {
		return fmt.Errorf("connect to runner via relay failed: %w", err)
	}

	return nil
}

func (c *Client) Stream(ctx context.Context, runnerInfo *peer.AddrInfo) error {
	s, err := c.Host.NewStream(network.WithAllowLimitedConn(context.Background(), "customprotocl"), runnerInfo.ID, protocol.ID("/customprotocol/1.0.0"))
	if err != nil {
		return fmt.Errorf("stream creation failed: %w", err)
	}
	defer s.Close()

	message := "Hello from client"
	if _, err := s.Write([]byte(message)); err != nil {
		return fmt.Errorf("write to stream failed: %w", err)
	}

	buf := make([]byte, 1024)
	n, err := s.Read(buf)
	if err != nil {
		return fmt.Errorf("read from stream failed: %w", err)
	}
	response := string(buf[:n])
	if response != "Hello from Runner" {
		return fmt.Errorf("unexpected response: %s", response)
	}

	return nil
}
