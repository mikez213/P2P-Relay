package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"

	logging "github.com/ipfs/go-log/v2"

	// create "mnwarm/internal/host"
	cmn "mnwarm/internal/shared"
)

var log = logging.Logger("bootlog")

func main() {
	logging.SetAllLoggers(logging.LevelError)
	logging.SetLogLevel("bootlog", "debug")

	portStr, keyIndexInt, bootstrapAddrs, err := cmn.ParseCmdArgs()
	nodeOpt, err := cmn.GetLibp2pIdentity(keyIndexInt)

	listenPort, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	// nodeType := create.Bootstrap
	//XLF2ipQ4jD3U
	// host, _, err := create.CreateHost(ctx, nodeType, nil, listenPort)
	host, err := libp2p.New(
		nodeOpt,
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		// libp2p.EnableRelay(),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableHolePunching(),
		libp2p.ForceReachabilityPublic(),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("bootstrap up pid %s", host.ID())
	log.Info("listening on:")
	for _, addr := range host.Addrs() {
		log.Infof("%s/p2p/%s", addr, host.ID())
	}

	kademliaDHT, err := dht.New(ctx, host, dht.Mode(dht.ModeServer))
	if err != nil {
		log.Fatal(err)
	}

	bootstrapPeers, err := cmn.ParseBootstrap(bootstrapAddrs)
	if len(bootstrapPeers) == 0 {
		log.Warn("no valid bootstrap addrs")
	}

	cmn.ConnectToBootstrapPeers(ctx, host, bootstrapPeers)
	cmn.BootstrapDHT(ctx, kademliaDHT)

	time.Sleep(2 * time.Second)

	log.Infof("running pid %s", host.ID())
	log.Info("use multiaddrs to connect:")
	for _, addr := range host.Addrs() {
		log.Infof("%s/p2p/%s", addr, host.ID())
	}

	// go func() {
	// 	for {
	// 		time.Sleep(30 * time.Second)
	// 		kademliaDHT.RefreshRoutingTable() //has a channel to block, but unused for now
	// 		peers := kademliaDHT.RoutingTable().ListPeers()
	// 		log.Infof("dht routing table peers (%d): %v", len(peers), peers)
	// 		log.Infof("network peers, %s, peerstore peers, %s", host.Network().Peers(), host.Peerstore().Peers())
	// 	}
	// }()

	select {}
}
