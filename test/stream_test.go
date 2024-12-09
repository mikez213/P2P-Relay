package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	cmn "mnwarm/internal/shared"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

func TestRelayStart(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Start relay successfully",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			relay, err := StartRelay(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartRelay() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				defer func() {
					if cerr := relay.Host.Close(); cerr != nil {
						t.Errorf("Close relay failed: %v", cerr)
					}
				}()
				t.Logf("Relay started with ID: %s", relay.Host.ID())
			}
		})
	}
}

func TestBootstrapPeers(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Connect to single relay peer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			relay, err := StartRelay(ctx)
			if err != nil {
				t.Fatalf("StartRelay failed: %v", err)
			}
			defer relay.Host.Close()

			if len(relay.Host.Addrs()) == 0 {
				t.Fatalf("Relay has no addresses")
			}

			bootstrap := []peer.AddrInfo{*relay.AddrInfo()}

			testHost, err := libp2p.New(
				libp2p.NoListenAddrs,
			)
			if err != nil {
				t.Fatalf("Create test host failed: %v", err)
			}
			defer testHost.Close()

			success, errs := cmn.ConnectToBootstrapPeers(ctx, testHost, bootstrap)
			if !success {
				t.Errorf("Expected successful connection, got errors: %v", errs)
			}
		})
	}
}

func TestDHTBootstrap(t *testing.T) {
	tests := []struct {
		name      string
		d         *dht.IpfsDHT
		wantErr   bool
		errorText string
	}{
		{
			name:      "Valid DHT bootstrap",
			d:         nil, // Will initialize
			wantErr:   false,
			errorText: "",
		},
		{
			name:      "Nil DHT bootstrap",
			d:         nil,
			wantErr:   true,
			errorText: "DHT not initialized properly",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var dhtInstance *dht.IpfsDHT

			if i == 0 {
				h, err := libp2p.New()
				if err != nil {
					t.Fatalf("Create host for DHT failed: %v", err)
				}
				defer h.Close()

				dhtInstance, err = dht.New(ctx, h)
				if err != nil {
					t.Fatalf("Create DHT failed: %v", err)
				}
				tt.d = dhtInstance
			} else {
				tt.d = nil
			}

			err := cmn.BootstrapDHT(ctx, tt.d)
			if (err != nil) != tt.wantErr {
				t.Errorf("BootstrapDHT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errorText) {
				t.Errorf("Error = %v, expected to contain %v", err, tt.errorText)
			}
		})
	}
}

func TestRelayReservation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(ctx context.Context) (host.Host, *peer.AddrInfo, error)
		wantErr bool
	}{
		{
			name: "Successful relay reservation",
			setup: func(ctx context.Context) (host.Host, *peer.AddrInfo, error) {
				relay, err := StartRelay(ctx)
				if err != nil {
					return nil, nil, fmt.Errorf("start relay failed: %w", err)
				}

				hostA, err := libp2p.New(
					libp2p.NoListenAddrs,
				)
				if err != nil {
					relay.Host.Close()
					return nil, nil, fmt.Errorf("create hostA failed: %w", err)
				}

				hostA.Peerstore().AddAddrs(relay.Host.ID(), relay.Host.Addrs(), peerstore.PermanentAddrTTL)
				return hostA, relay.AddrInfo(), nil
			},
			wantErr: false,
		},
		{
			name: "Relay reservation with nil info",
			setup: func(ctx context.Context) (host.Host, *peer.AddrInfo, error) {
				hostB, err := libp2p.New(
					libp2p.NoListenAddrs,
				)
				if err != nil {
					return nil, nil, fmt.Errorf("create hostB failed: %w", err)
				}
				return hostB, nil, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			host, relayInfo, err := tt.setup(ctx)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			defer host.Close()

			err = cmn.ReserveRelay(ctx, host, relayInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReserveRelay() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunnerRelayConnection(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Runner connects to relay",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			relay, err := StartRelay(ctx)
			if err != nil {
				t.Fatalf("StartRelay failed: %v", err)
			}
			defer relay.Host.Close()

			runner, err := StartRunner(ctx, relay.AddrInfo())
			if (err != nil) != tt.wantErr {
				t.Errorf("StartRunner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				defer runner.Host.Close()

				connected := runner.Host.Network().Connectedness(relay.Host.ID()) == network.Connected
				if !connected {
					t.Fatal("Runner is not connected to relay")
				}
				t.Logf("Runner connected to relay successfully")
			}
		})
	}
}

func TestClientRelayConnection(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Client connects to relay",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			relay, err := StartRelay(ctx)
			if err != nil {
				t.Fatalf("StartRelay failed: %v", err)
			}
			defer relay.Host.Close()

			client, err := StartClient(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				defer client.Host.Close()

				client.Host.Peerstore().AddAddrs(relay.Host.ID(), relay.Host.Addrs(), peerstore.PermanentAddrTTL)

				err := cmn.ConnectToRelay(ctx, client.Host, relay.AddrInfo())
				if err != nil {
					t.Fatalf("ConnectToRelay failed: %v", err)
				}

				connected := client.Host.Network().Connectedness(relay.Host.ID()) == network.Connected
				if !connected {
					t.Fatal("Client is not connected to relay")
				}

				t.Logf("Client connected to relay successfully")
			}
		})
	}
}

func TestClientRunnerCommunication(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Client communicates with runner",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			relay, err := StartRelay(ctx)
			if err != nil {
				t.Fatalf("StartRelay failed: %v", err)
			}
			defer relay.Host.Close()

			runner, err := StartRunner(ctx, relay.AddrInfo())
			if err != nil {
				t.Fatalf("StartRunner failed: %v", err)
			}
			defer runner.Host.Close()

			client, err := StartClient(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				defer client.Host.Close()

				if err := client.Connect(ctx, relay.AddrInfo(), runner.AddrInfo()); err != nil {
					t.Fatalf("Client initial connection failed: %v", err)
				}

				connectedness := client.Host.Network().Connectedness(runner.Host.ID())
				if connectedness != network.Connected && connectedness != network.Limited {
					t.Fatalf("Client not connected to runner, status: %s", connectedness)
				}

				t.Logf("Client connected to runner via relay, status: %s", connectedness)
			}
		})
	}
}

func TestStream(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Client communicates with runner",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			relay, err := StartRelay(ctx)
			if err != nil {
				t.Fatalf("StartRelay failed: %v", err)
			}
			defer relay.Host.Close()

			runner, err := StartRunner(ctx, relay.AddrInfo())
			if err != nil {
				t.Fatalf("StartRunner failed: %v", err)
			}
			defer runner.Host.Close()

			client, err := StartClient(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				defer client.Host.Close()

				if err := client.Connect(ctx, relay.AddrInfo(), runner.AddrInfo()); err != nil {
					t.Fatalf("Client initial connection failed: %v", err)
				}

				time.Sleep(2 * time.Second)

				if err := client.Stream(ctx, runner.AddrInfo()); (err != nil) != tt.wantErr {
					t.Errorf("Stream() error = %v, wantErr %v", err, tt.wantErr)
				} else if err == nil {
					t.Log("Client successfully communicated with runner")
				}
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Complete integration test",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			relay, err := StartRelay(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer relay.Host.Close()

			runner, err := StartRunner(ctx, relay.AddrInfo())
			if err != nil {
				t.Fatal(err)
			}
			defer runner.Host.Close()

			client, err := StartClient(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Host.Close()

			if err := client.Connect(ctx, relay.AddrInfo(), runner.AddrInfo()); err != nil {
				t.Fatalf("Client initial connection failed: %v", err)
			}

			time.Sleep(5 * time.Second)

			if err := client.Stream(ctx, runner.AddrInfo()); (err != nil) != tt.wantErr {
				t.Errorf("Stream() error = %v, wantErr %v", err, tt.wantErr)
			} else if err == nil {
				t.Log("Integration test completed successfully")
			}
		})
	}
}
