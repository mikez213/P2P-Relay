// integrated_test.go
package tests

import (
	"context"
	cmn "mnwarm/internal/shared"
	"testing"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestRelayNodeStart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	relayNode, err := StartRelayNode(ctx)
	if err != nil {
		t.Fatalf("Failed to start relay node: %v", err)
	}
	defer func() {
		if err := relayNode.Host.Close(); err != nil {
			t.Errorf("Error closing relay node host: %v", err)
		}
	}()

	t.Logf("Relay node started successfully with ID: %s", relayNode.Host.ID())
}

func TestConnectToBootstrapPeers(t *testing.T) {
	ctx := context.Background()
	h, err := libp2p.New()
	if err != nil {
		t.Fatalf("Failed to create mock host: %v", err)
	}
	defer h.Close()

	testPeerID, err := peer.Decode("12D3KooWLr1gYejUTeriAsSu6roR2aQ423G3Q4fFTqzqSwTsMz9n")
	if err != nil {
		t.Fatalf("Failed to decode test peer ID: %v", err)
	}
	bootstrapPeers := []peer.AddrInfo{
		{ID: testPeerID},
	}

	type args struct {
		ctx            context.Context
		host           host.Host
		bootstrapPeers []peer.AddrInfo
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Connect to bootstrap peers",
			args: args{
				ctx:            ctx,
				host:           h,
				bootstrapPeers: bootstrapPeers,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmn.ConnectToBootstrapPeers(tt.args.ctx, tt.args.host, tt.args.bootstrapPeers)
		})
	}
}

func TestBootstrapDHT(t *testing.T) {
	ctx := context.Background()
	h, err := libp2p.New()
	if err != nil {
		t.Fatalf("Failed to create mock host: %v", err)
	}
	defer h.Close()

	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		t.Fatalf("Failed to create DHT: %v", err)
	}

	type args struct {
		ctx         context.Context
		kademliaDHT *dht.IpfsDHT
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Bootstrap DHT",
			args: args{
				ctx:         ctx,
				kademliaDHT: kademliaDHT,
			},
		},
		{
			name: "Nil DHT",
			args: args{
				ctx:         ctx,
				kademliaDHT: nil,
			},
		},
	}
	for _, tt := range tests {
		if tt.name == "Nil DHT" {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("BootstrapDHT() did not panic with nil DHT")
				}
			}()
		}
		t.Run(tt.name, func(t *testing.T) {
			cmn.BootstrapDHT(tt.args.ctx, tt.args.kademliaDHT)
		})
	}
}

func TestNodeRunnerConnectsToRelay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	relayNode, err := StartRelayNode(ctx)
	if err != nil {
		t.Fatalf("Failed to start relay node: %v", err)
	}
	defer func() {
		if err := relayNode.Host.Close(); err != nil {
			t.Errorf("Error closing relay node host: %v", err)
		}
	}()

	nodeRunner, err := StartNodeRunner(ctx, relayNode.AddrInfo())
	if err != nil {
		t.Fatalf("Failed to start node runner: %v", err)
	}
	defer func() {
		if err := nodeRunner.Host.Close(); err != nil {
			t.Errorf("Error closing node runner host: %v", err)
		}
	}()

	connected := nodeRunner.Host.Network().Connectedness(relayNode.Host.ID()) == network.Connected
	if !connected {
		t.Fatal("Node runner is not connected to relay node")
	}

	t.Logf("Node runner connected to relay node successfully")
}

func TestMobileClientConnectsToRelay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	relayNode, err := StartRelayNode(ctx)
	if err != nil {
		t.Fatalf("Failed to start relay node: %v", err)
	}
	defer func() {
		if err := relayNode.Host.Close(); err != nil {
			t.Errorf("Error closing relay node host: %v", err)
		}
	}()

	mobileClient, err := StartMobileClient(ctx)
	if err != nil {
		t.Fatalf("Failed to start mobile client: %v", err)
	}
	defer func() {
		if err := mobileClient.Host.Close(); err != nil {
			t.Errorf("Error closing mobile client host: %v", err)
		}
	}()

	err = cmn.ConnectToRelay(ctx, mobileClient.Host, relayNode.AddrInfo())
	if err != nil {
		t.Fatalf("Mobile client failed to connect to relay: %v", err)
	}

	connected := mobileClient.Host.Network().Connectedness(relayNode.Host.ID()) == network.Connected
	if !connected {
		t.Fatal("Mobile client is not connected to relay node")
	}

	t.Logf("Mobile client connected to relay node successfully")
}

func TestMobileClientConnectsToNodeRunner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	relayNode, err := StartRelayNode(ctx)
	if err != nil {
		t.Fatalf("Failed to start relay node: %v", err)
	}
	defer func() {
		if err := relayNode.Host.Close(); err != nil {
			t.Errorf("Error closing relay node host: %v", err)
		}
	}()

	nodeRunner, err := StartNodeRunner(ctx, relayNode.AddrInfo())
	if err != nil {
		t.Fatalf("Failed to start node runner: %v", err)
	}
	defer func() {
		if err := nodeRunner.Host.Close(); err != nil {
			t.Errorf("Error closing node runner host: %v", err)
		}
	}()

	mobileClient, err := StartMobileClient(ctx)
	if err != nil {
		t.Fatalf("Failed to start mobile client: %v", err)
	}
	defer func() {
		if err := mobileClient.Host.Close(); err != nil {
			t.Errorf("Error closing mobile client host: %v", err)
		}
	}()

	err = mobileClient.InitialConnection(ctx, relayNode.AddrInfo(), nodeRunner.AddrInfo())
	if err != nil {
		t.Fatalf("Mobile client initial connection failed: %v", err)
	}

	connectedness := mobileClient.Host.Network().Connectedness(nodeRunner.Host.ID())
	if connectedness != network.Connected && connectedness != network.Limited {
		t.Fatalf("Mobile client is not connected to node runner, connected level is %s", connectedness)
	}

	t.Logf("Mobile client connected to node runner via relay successfully, connected level is %s", connectedness)
}

func TestMobileClientStreamNodeRunner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	relayNode, err := StartRelayNode(ctx)
	if err != nil {
		t.Fatalf("Failed to start relay node: %v", err)
	}
	defer func() {
		if err := relayNode.Host.Close(); err != nil {
			t.Errorf("Error closing relay node host: %v", err)
		}
	}()

	nodeRunner, err := StartNodeRunner(ctx, relayNode.AddrInfo())
	if err != nil {
		t.Fatalf("Failed to start node runner: %v", err)
	}
	defer func() {
		if err := nodeRunner.Host.Close(); err != nil {
			t.Errorf("Error closing node runner host: %v", err)
		}
	}()

	mobileClient, err := StartMobileClient(ctx)
	if err != nil {
		t.Fatalf("Failed to start mobile client: %v", err)
	}
	defer func() {
		if err := mobileClient.Host.Close(); err != nil {
			t.Errorf("Error closing mobile client host: %v", err)
		}
	}()

	err = mobileClient.InitialConnection(ctx, relayNode.AddrInfo(), nodeRunner.AddrInfo())
	if err != nil {
		t.Fatalf("Mobile client initial connection failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	err = mobileClient.CommunicateWithNodeRunner(ctx, nodeRunner.AddrInfo())
	if err != nil {
		t.Fatalf("Communication with node runner failed: %v", err)
	}

	t.Log("Mobile client successfully communicated with node runner")
}

func TestV2(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	relayNode, err := StartRelayNode(ctx)
	if err != nil {
		t.Error(err)
	}
	nodeRunner, err := StartNodeRunner(ctx, relayNode.AddrInfo())
	if err != nil {
		t.Error(err)
	}
	mobileClient, err := StartMobileClient(ctx)
	if err != nil {
		t.Error(err)
	}

	err = mobileClient.InitialConnection(ctx, relayNode.AddrInfo(), nodeRunner.AddrInfo())
	if err != nil {
		t.Fatalf("Mobile client initial connection failed: %v", err)
	}
	time.Sleep(5 * time.Second)

	mobileClient.CommunicateWithNodeRunnerOld(t, relayNode.AddrInfo(), nodeRunner.AddrInfo())
	t.Log("test ok")
}
