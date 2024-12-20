package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("sharedlog")

var BootstrapPeerIDs = []peer.ID{}

var Shearing = "hehele"
var private = "private"

var RelayerPrivateKeys = []string{
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
	// 12D3KooWNS4QQxwNURwoYoXmGjH9AQkagcGTjRUQT33P4i4FKQsi
	"CAESQCSHrfyzNZkxwoNmXI1wx5Lvr6o4+kGxGepFH0AfYlKthyON+1hQRjLJQaBAQLrr1cfMHFFoC40X62DQIhL246U=",
	// 12D3KooWJuteouY1d5SYFcAUAYDVPjFD8MUBgqsdjZfBkAecCS2Y
	"CAESQDyiSqC9Jez8wKSQs74YJalAegamjVKHbnaN35pfe6Gk21WVgCzfvBdLVoRj8XXny/k1LtSOhPZWNz0rWKCOYpk=",
	// 12D3KooWQaZ9Ppi8A2hcEspJhewfPqKjtXu4vx7FQPaUGnHXWpNL
}

func RelayIdentity(keyIndex int) (libp2p.Option, error) {
	if keyIndex < 0 || keyIndex >= len(RelayerPrivateKeys) {
		return nil, fmt.Errorf("invalid key index: %d", keyIndex)
	}

	keyStr := RelayerPrivateKeys[keyIndex]
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

func GetLibp2pIdentity(keyIndex int) (libp2p.Option, error) {
	nodeOpt, err := RelayIdentity(keyIndex)
	if err != nil {
		return nil, fmt.Errorf("relay identity error: %w", err)
	}
	log.Debugf("identity is %+v", nodeOpt)
	return nodeOpt, nil
}

func IsBootstrapPeer(peerID peer.ID) bool {
	for _, bootstrapID := range BootstrapPeerIDs {
		if peerID == bootstrapID {
			return true
		}
	}
	return false
}

func ContainsPeer(relayAddresses []peer.AddrInfo, pid peer.ID) bool {
	for _, relayAddrInfo := range relayAddresses {
		if relayAddrInfo.ID == pid {
			return true
		}
	}
	return false
}

func IsInvalidTarget(relayAddresses []peer.AddrInfo, pid peer.ID) bool {
	return (IsBootstrapPeer(pid) || ContainsPeer(relayAddresses, pid))
}

func ParseBootstrap(bootstrapAddrs []string) ([]peer.AddrInfo, error) {
	var bootstrapPeers []peer.AddrInfo
	var parseErrors []string

	for _, addrStr := range bootstrapAddrs {
		addrStr = strings.TrimSpace(addrStr)
		if addrStr == "" {
			continue
		}
		maddr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			errMsg := fmt.Sprintf("invalid bootstrap addr '%s': %v", addrStr, err)
			log.Error(errMsg)
			parseErrors = append(parseErrors, errMsg)
			continue
		}
		peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			errMsg := fmt.Sprintf("failed to parse bootstrap peer info from '%s': %v", addrStr, err)
			log.Error(errMsg)
			parseErrors = append(parseErrors, errMsg)
			continue
		}
		bootstrapPeers = append(bootstrapPeers, *peerInfo)
	}

	if len(parseErrors) > 0 {
		return bootstrapPeers, fmt.Errorf("encountered errors while parsing bootstrap addresses: %v", parseErrors)
	}

	return bootstrapPeers, nil
}

func ParseCmdArgs() (string, int, []string, error) {
	if len(os.Args) < 3 {
		errMsg := "need a bootstrap node and relay"
		log.Error(errMsg)
		return "", 0, nil, errors.New(errMsg)
	}

	relayAddrStr := os.Args[1]
	keyIndexStr := os.Args[2]
	bootstrapAddrs := os.Args[3:]

	keyIndexInt, err := strconv.Atoi(keyIndexStr)
	if err != nil {
		errMsg := fmt.Sprintf("index error: %v", err)
		log.Error(errMsg)
		return "", 0, nil, errors.New(errMsg)
	}

	return relayAddrStr, keyIndexInt, bootstrapAddrs, nil
}

func ParseRelayAddress(relayAddrStr string) (*peer.AddrInfo, error) {
	relayMaddr, err := multiaddr.NewMultiaddr(relayAddrStr)
	if err != nil {
		errMsg := fmt.Sprintf("bad relay address '%s': %v", relayAddrStr, err)
		log.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	relayInfo, err := peer.AddrInfoFromP2pAddr(relayMaddr)
	if err != nil {
		errMsg := fmt.Sprintf("fail to parse relay peer info from '%s': %v", relayAddrStr, err)
		log.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	log.Infof("relay info: %s, address: %v", relayInfo.ID, relayInfo.Addrs)

	return relayInfo, nil
}

func AssembleRelay(relayAddrInfo peer.AddrInfo, p peer.AddrInfo) (peer.AddrInfo, error) {
	if len(relayAddrInfo.Addrs) == 0 {
		errMsg := fmt.Sprintf("relay %s has no addresses!!!!", relayAddrInfo.ID)
		log.Error(errMsg)
		return peer.AddrInfo{}, fmt.Errorf(errMsg)
	}

	relayAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p-circuit/p2p/%s", p.ID))
	if err != nil {
		errMsg := fmt.Sprintf("failed to create relay circuit multiaddr: %v", err)
		log.Error(errMsg)
		return peer.AddrInfo{}, fmt.Errorf(errMsg)
	}

	combinedRelayAddr := relayAddrInfo.Addrs[0].Encapsulate(relayAddr)
	p.Addrs = append(p.Addrs, combinedRelayAddr)

	log.Infof("trying to connect to peer %s via relay %s", p.ID, relayAddrInfo.ID)
	log.Infof("relay address: %s", combinedRelayAddr)

	targetInfo, err := peer.AddrInfoFromP2pAddr(combinedRelayAddr)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create peer.AddrInfo from combined relay address '%s': %v", combinedRelayAddr, err)
		log.Error(errMsg)
		return peer.AddrInfo{}, fmt.Errorf(errMsg)
	}
	targetID := targetInfo.ID

	newRelayAddr, err := multiaddr.NewMultiaddr("/p2p/" + relayAddrInfo.ID.String() + "/p2p-circuit/p2p/" + targetID.String())
	if err != nil {
		errMsg := fmt.Sprintf("failed to create new relay multiaddr: %v", err)
		log.Error(errMsg)
		return peer.AddrInfo{}, fmt.Errorf(errMsg)
	}

	log.Infof("newRelayAddr: %v", newRelayAddr)

	targetRelayedInfo := peer.AddrInfo{
		ID:    targetID,
		Addrs: []multiaddr.Multiaddr{newRelayAddr},
	}

	log.Infof("targetRelayedInfo: %v", targetRelayedInfo)

	return targetRelayedInfo, nil
}

func ConnectToBootstrapPeers(ctx context.Context, host host.Host, bootstrapPeers []peer.AddrInfo) (bool, []error) {
	var success bool
	var errs []error

	for _, peerInfo := range bootstrapPeers {
		log.Infof("connecting to bootstrap node %s", peerInfo.ID)
		if err := host.Connect(ctx, peerInfo); err != nil {
			errMsg := fmt.Sprintf("Failed to connect to bootstrap node %s: %v", peerInfo.ID, err)
			log.Error(errMsg)
			errs = append(errs, fmt.Errorf("bootstrap node %s: %w", peerInfo.ID, err))
			continue
		} else {
			log.Infof("connected to bootstrap node %s", peerInfo.ID)
			success = true
		}
	}

	if !success {
		errMsg := "failed to connect to any bootstrap nodes"
		log.Error(errMsg)
		errs = append(errs, errors.New(errMsg))
	}

	return success, errs
}

func BootstrapDHT(ctx context.Context, kademliaDHT *dht.IpfsDHT) error {
	if kademliaDHT == nil {
		errMsg := "DHT not initialized properly"
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	if err := kademliaDHT.Bootstrap(ctx); err != nil {
		errMsg := fmt.Sprintf("failed to bootstrap DHT: %v", err)
		log.Error(errMsg)
		return fmt.Errorf("DHT bootstrap error: %w", err)
	}

	log.Info("DHT bootstrapped successfully")
	return nil
}

func ConnectToRelay(ctx context.Context, host host.Host, relayInfo *peer.AddrInfo) error {
	if relayInfo == nil {
		errMsg := "relayInfo is nil"
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	log.Infof("connecting to relay node %s", relayInfo.ID)
	if err := host.Connect(ctx, *relayInfo); err != nil {
		errMsg := fmt.Sprintf("failed to connect to relay node %s: %v", relayInfo.ID, err)
		log.Error(errMsg)
		return fmt.Errorf("relay node %s connection error: %w", relayInfo.ID, err)
	}
	log.Infof("connected to relay node %s", relayInfo.ID)
	return nil
}

func ConstructRelayAddresses(host host.Host, relayInfo *peer.AddrInfo) ([]peer.AddrInfo, error) {
	var relayAddresses []peer.AddrInfo
	var constructionErrors []string

	for _, addr := range relayInfo.Addrs {
		fullRelayAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p-circuit/p2p/%s", relayInfo.ID))
		if err != nil {
			errMsg := fmt.Sprintf("failed to create relay circuit multiaddr: %v", err)
			log.Error(errMsg)
			constructionErrors = append(constructionErrors, errMsg)
			continue
		}
		log.Infof("created relay circuit multiaddr: %s", fullRelayAddr)

		combinedAddr := addr.Encapsulate(fullRelayAddr)
		log.Infof("final addr: %s", combinedAddr)

		relayAddrInfo := peer.AddrInfo{
			ID:    relayInfo.ID,
			Addrs: []multiaddr.Multiaddr{combinedAddr},
		}

		relayAddresses = append(relayAddresses, relayAddrInfo)
	}

	if len(constructionErrors) > 0 {
		return relayAddresses, fmt.Errorf("errors occurred while constructing relay addresses: %v", constructionErrors)
	}

	log.Infof("we are hopefully listening on the following relay addresses:")
	for _, addrInfo := range relayAddresses {
		for _, addr := range addrInfo.Addrs {
			log.Infof("%s/p2p/%s", addr, host.ID())
		}
	}

	return relayAddresses, nil
}

func ReserveRelay(ctx context.Context, host host.Host, relayInfo *peer.AddrInfo) error {
	if relayInfo == nil {
		errMsg := "relayInfo is nil"
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	cli, err := client.Reserve(ctx, host, *relayInfo)
	if err != nil {
		errMsg := fmt.Sprintf("failed to receive a relay reservation from relay: %v", err)
		log.Error(errMsg)
		return fmt.Errorf("relay reservation error: %w", err)
	}
	log.Infof("Relay reservation details: %+v", cli)
	log.Info("relay reservation successful")
	return nil
}
