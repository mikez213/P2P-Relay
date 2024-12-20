package common

import (
	"strings"
	"testing"

	crypto "github.com/libp2p/go-libp2p/core/crypto"
	peer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func TestRelayIdentity(t *testing.T) {
	originalRelayerPrivateKeys := RelayerPrivateKeys
	defer func() { RelayerPrivateKeys = originalRelayerPrivateKeys }()

	privKey1, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		t.Fatalf("Failed to generate test private key 1: %v", err)
	}
	privKeyBytes1, err := crypto.MarshalPrivateKey(privKey1)
	if err != nil {
		t.Fatalf("Failed to marshal private key 1: %v", err)
	}
	encodedPrivKey1 := crypto.ConfigEncodeKey(privKeyBytes1)

	privKey2, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		t.Fatalf("Failed to generate test private key 2: %v", err)
	}
	privKeyBytes2, err := crypto.MarshalPrivateKey(privKey2)
	if err != nil {
		t.Fatalf("Failed to marshal private key 2: %v", err)
	}
	encodedPrivKey2 := crypto.ConfigEncodeKey(privKeyBytes2)

	RelayerPrivateKeys = []string{encodedPrivKey1, encodedPrivKey2}

	type args struct {
		keyIndex int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Valid keyIndex 0",
			args:    args{keyIndex: 0},
			wantErr: false,
		},
		{
			name:    "Valid keyIndex 1",
			args:    args{keyIndex: 1},
			wantErr: false,
		},
		{
			name:    "Invalid keyIndex -1",
			args:    args{keyIndex: -1},
			wantErr: true,
		},
		{
			name:    "Invalid keyIndex out of range",
			args:    args{keyIndex: 2},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RelayIdentity(tt.args.keyIndex)
			if (err != nil) != tt.wantErr {
				t.Errorf("RelayIdentity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGetLibp2pIdentity(t *testing.T) {
	originalRelayerPrivateKeys := RelayerPrivateKeys
	defer func() { RelayerPrivateKeys = originalRelayerPrivateKeys }()

	privKey1, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		t.Fatalf("Failed to generate test private key 1: %v", err)
	}
	privKeyBytes1, err := crypto.MarshalPrivateKey(privKey1)
	if err != nil {
		t.Fatalf("Failed to marshal private key 1: %v", err)
	}
	encodedPrivKey1 := crypto.ConfigEncodeKey(privKeyBytes1)

	RelayerPrivateKeys = []string{encodedPrivKey1}

	type args struct {
		keyIndex int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Valid keyIndex 0",
			args:    args{keyIndex: 0},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetLibp2pIdentity(tt.args.keyIndex)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLibp2pIdentity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsBootstrapPeer(t *testing.T) {
	originalBootstrapPeerIDs := BootstrapPeerIDs
	defer func() { BootstrapPeerIDs = originalBootstrapPeerIDs }()

	testPeerID1, err := peer.Decode("12D3KooWBnext3VBZZuBwGn3YahAZjf49oqYckfx64VpzH6dyU1p")
	if err != nil {
		t.Fatalf("Failed to decode test peer ID 1: %v", err)
	}
	testPeerID2, err := peer.Decode("12D3KooWLr1gYejUTeriAsSu6roR2aQ423G3Q4fFTqzqSwTsMz9n")
	if err != nil {
		t.Fatalf("Failed to decode test peer ID 2: %v", err)
	}

	BootstrapPeerIDs = []peer.ID{testPeerID1, testPeerID2}

	type args struct {
		peerID peer.ID
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Is bootstrap peer - testPeerID1",
			args: args{peerID: testPeerID1},
			want: true,
		},
		{
			name: "Is bootstrap peer - testPeerID2",
			args: args{peerID: testPeerID2},
			want: true,
		},
		{
			name: "Not a bootstrap peer",
			args: args{peerID: peer.ID("12D3KooWNotBootstrapPeer")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBootstrapPeer(tt.args.peerID); got != tt.want {
				t.Errorf("IsBootstrapPeer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsPeer(t *testing.T) {
	testPeerID1, err := peer.Decode("12D3KooWRnBKUEkAEpsoCoEiuhxKBJ5j2Bdop6PGxFMvd4PwoevM")
	if err != nil {
		t.Fatalf("Failed to decode test peer ID 1: %v", err)
	}
	testPeerID2, err := peer.Decode("12D3KooWRgSQnguL2DYkXUXqCLiRQ35PEX4eEH3havy2X18AVALd")
	if err != nil {
		t.Fatalf("Failed to decode test peer ID 2: %v", err)
	}

	relayAddresses := []peer.AddrInfo{
		{ID: testPeerID1},
		{ID: testPeerID2},
	}

	type args struct {
		relayAddresses []peer.AddrInfo
		pid            peer.ID
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Contains peer - testPeerID1",
			args: args{
				relayAddresses: relayAddresses,
				pid:            testPeerID1,
			},
			want: true,
		},
		{
			name: "Contains peer - testPeerID2",
			args: args{
				relayAddresses: relayAddresses,
				pid:            testPeerID2,
			},
			want: true,
		},
		{
			name: "Does not contain peer",
			args: args{
				relayAddresses: relayAddresses,
				pid:            peer.ID("12D3KooWQaZ9Ppi8A2hcEspJhewfPqKjtXu4vx7FQPaUGnHXWpNL"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsPeer(tt.args.relayAddresses, tt.args.pid); got != tt.want {
				t.Errorf("ContainsPeer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInvalidTarget(t *testing.T) {
	originalBootstrapPeerIDs := BootstrapPeerIDs
	defer func() { BootstrapPeerIDs = originalBootstrapPeerIDs }()

	bootstrapPeerID, err := peer.Decode("12D3KooWLr1gYejUTeriAsSu6roR2aQ423G3Q4fFTqzqSwTsMz9n")
	if err != nil {
		t.Fatalf("Failed to decode bootstrap peer ID: %v", err)
	}
	BootstrapPeerIDs = []peer.ID{bootstrapPeerID}

	relayPeerID, err := peer.Decode("12D3KooWRnBKUEkAEpsoCoEiuhxKBJ5j2Bdop6PGxFMvd4PwoevM")
	if err != nil {
		t.Fatalf("Failed to decode relay peer ID: %v", err)
	}
	relayAddresses := []peer.AddrInfo{
		{ID: relayPeerID},
	}

	type args struct {
		relayAddresses []peer.AddrInfo
		pid            peer.ID
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Is invalid target - Bootstrap peer",
			args: args{
				relayAddresses: relayAddresses,
				pid:            bootstrapPeerID,
			},
			want: true,
		},
		{
			name: "Is invalid target - Relay peer",
			args: args{
				relayAddresses: relayAddresses,
				pid:            relayPeerID,
			},
			want: true,
		},
		{
			name: "Valid target",
			args: args{
				relayAddresses: relayAddresses,
				pid:            peer.ID("12D3KooWNS4QQxwNURwoYoXmGjH9AQkagcGTjRUQT33P4i4FKQsi"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInvalidTarget(tt.args.relayAddresses, tt.args.pid); got != tt.want {
				t.Errorf("IsInvalidTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBootstrap(t *testing.T) {
	validAddr1 := "/ip4/127.0.0.1/tcp/1234/p2p/12D3KooWLr1gYejUTeriAsSu6roR2aQ423G3Q4fFTqzqSwTsMz9n"
	validAddr2 := "/ip4/127.0.0.1/tcp/5678/p2p/12D3KooWBnext3VBZZuBwGn3YahAZjf49oqYckfx64VpzH6dyU1p"
	invalidAddr := "/invalid/multiaddr"

	pid1, err := peer.Decode("12D3KooWLr1gYejUTeriAsSu6roR2aQ423G3Q4fFTqzqSwTsMz9n")
	if err != nil {
		t.Fatalf("Failed to decode peer ID 1: %v", err)
	}
	pid2, err := peer.Decode("12D3KooWBnext3VBZZuBwGn3YahAZjf49oqYckfx64VpzH6dyU1p")
	if err != nil {
		t.Fatalf("Failed to decode peer ID 2: %v", err)
	}

	type args struct {
		bootstrapAddrs []string
	}
	tests := []struct {
		name    string
		args    args
		want    []peer.AddrInfo
		wantErr bool
	}{
		{
			name: "Valid addresses",
			args: args{bootstrapAddrs: []string{validAddr1, validAddr2}},
			want: []peer.AddrInfo{
				{ID: pid1},
				{ID: pid2},
			},
			wantErr: false,
		},
		{
			name: "Contains invalid address",
			args: args{bootstrapAddrs: []string{validAddr1, invalidAddr}},
			want: []peer.AddrInfo{
				{ID: pid1},
			},
			wantErr: true,
		},
		{
			name:    "Empty addresses",
			args:    args{bootstrapAddrs: []string{}},
			want:    []peer.AddrInfo{},
			wantErr: false,
		},
		{
			name:    "Only invalid addresses",
			args:    args{bootstrapAddrs: []string{invalidAddr}},
			want:    []peer.AddrInfo{},
			wantErr: true,
		},
		{
			name: "Mixed valid and empty addresses",
			args: args{bootstrapAddrs: []string{"", validAddr2, "   "}},
			want: []peer.AddrInfo{
				{ID: pid2},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBootstrap(tt.args.bootstrapAddrs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseBootstrap() got %v addresses, want %v addresses", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i].ID != tt.want[i].ID {
					t.Errorf("ParseBootstrap() got ID %v, want ID %v", got[i].ID, tt.want[i].ID)
				}
			}
		})
	}
}
func TestParseRelayAddress(t *testing.T) {
	validPeerIDStr := "12D3KooWRnBKUEkAEpsoCoEiuhxKBJ5j2Bdop6PGxFMvd4PwoevM"
	validAddr := "/ip4/127.0.0.1/tcp/1234/p2p/" + validPeerIDStr
	invalidAddr := "/invalid/multiaddr"

	validPeerID, err := peer.Decode(validPeerIDStr)
	if err != nil {
		t.Fatalf("Failed to decode valid peer ID: %v", err)
	}

	expectedAddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/1234")
	if err != nil {
		t.Fatalf("Failed to create expected multiaddr: %v", err)
	}

	type args struct {
		relayAddrStr string
	}
	tests := []struct {
		name    string
		args    args
		want    *peer.AddrInfo
		wantErr bool
		errText string
	}{
		{
			name:    "Valid relay address",
			args:    args{relayAddrStr: validAddr},
			want:    &peer.AddrInfo{ID: validPeerID, Addrs: []multiaddr.Multiaddr{expectedAddr}},
			wantErr: false,
		},
		{
			name:    "Invalid relay address",
			args:    args{relayAddrStr: invalidAddr},
			want:    nil,
			wantErr: true,
			errText: "bad relay address",
		},
		{
			name:    "Empty relay address",
			args:    args{relayAddrStr: ""},
			want:    nil,
			wantErr: true,
			errText: "bad relay address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRelayAddress(tt.args.relayAddrStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRelayAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ParseRelayAddress() error = %v, expected to contain %v", err, tt.errText)
			}
			if !tt.wantErr && got != nil && got.ID != tt.want.ID {
				t.Errorf("ParseRelayAddress() got ID = %v, want ID = %v", got.ID, tt.want.ID)
			}
		})
	}
}
func TestAssembleRelay(t *testing.T) {
	relayPeerID, err := peer.Decode("12D3KooWRnBKUEkAEpsoCoEiuhxKBJ5j2Bdop6PGxFMvd4PwoevM")
	if err != nil {
		t.Fatalf("Failed to decode relay peer ID: %v", err)
	}
	targetPeerID, err := peer.Decode("12D3KooWNS4QQxwNURwoYoXmGjH9AQkagcGTjRUQT33P4i4FKQsi")
	if err != nil {
		t.Fatalf("Failed to decode target peer ID: %v", err)
	}

	relayMaddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/1234")
	if err != nil {
		t.Fatalf("Failed to create relay multiaddr: %v", err)
	}

	targetMaddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/5678")
	if err != nil {
		t.Fatalf("Failed to create target multiaddr: %v", err)
	}

	relayAddrInfo := peer.AddrInfo{
		ID:    relayPeerID,
		Addrs: []multiaddr.Multiaddr{relayMaddr},
	}

	targetAddrInfo := peer.AddrInfo{
		ID:    targetPeerID,
		Addrs: []multiaddr.Multiaddr{targetMaddr},
	}

	type args struct {
		relayAddrInfo peer.AddrInfo
		p             peer.AddrInfo
	}
	tests := []struct {
		name    string
		args    args
		want    peer.AddrInfo
		wantErr bool
	}{
		{
			name: "Valid relay and target",
			args: args{
				relayAddrInfo: relayAddrInfo,
				p:             targetAddrInfo,
			},
			wantErr: false,
		},
		{
			name: "Relay with no addresses",
			args: args{
				relayAddrInfo: peer.AddrInfo{
					ID:    relayPeerID,
					Addrs: []multiaddr.Multiaddr{},
				},
				p: targetAddrInfo,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AssembleRelay(tt.args.relayAddrInfo, tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssembleRelay() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got.ID != tt.args.p.ID {
				t.Errorf("AssembleRelay() got ID = %v, want ID = %v", got.ID, tt.args.p.ID)
			}
		})
	}
}
