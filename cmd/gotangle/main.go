package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
    "github.com/libp2p/go-libp2p/p2p/protocol/ping"
	peerstore "github.com/libp2p/go-libp2p/core/peer"
	routing "github.com/libp2p/go-libp2p/core/routing"
	kad "github.com/libp2p/go-libp2p-kad-dht"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	ma "github.com/multiformats/go-multiaddr"
)

// mdnsNotifee receives discovered peers from mdns and connects to them.
type mdnsNotifee struct {
	h host.Host
}

func (n *mdnsNotifee) HandlePeerFound(pi peerstore.AddrInfo) {
	fmt.Printf("mDNS discovered: %s\n", pi.ID.String())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := n.h.Connect(ctx, pi); err != nil {
		fmt.Printf("Error connecting to discovered peer %s: %s\n", pi.ID.String(), err)
	}
}

func main() {
	port := flag.Int("port", 0, "port to listen on (0 = random)")
	peerAddr := flag.String("peer", "", "peer multiaddress to connect to")

	flag.Parse()

	ctx := context.Background()

	// generate identity
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}

	// create host
	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port)),
	)
	identify.NewIDService(h)
	ping.NewPingService(h)
	if err != nil {
		log.Fatalf("Failed to create libp2p host: %v", err)
	}

	fmt.Printf("Hello! I am a libp2p node. Peer ID: %s\n", h.ID().String())
	for _, a := range h.Addrs() {
		fmt.Printf(" Listening on: %s/p2p/%s\n", a.String(), h.ID().String())
	}

	// If a peer address was provided, try to connect manually
	if *peerAddr != "" {
		fmt.Printf("Attempting to connect to peer: %s\n", *peerAddr)
		maddr, err := ma.NewMultiaddr(*peerAddr)
		if err != nil {
			log.Fatalf("Invalid peer multiaddress: %v", err)
		}

		info, err := peerstore.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Fatalf("Failed to parse peer info: %v", err)
		}

		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := h.Connect(ctx, *info); err != nil {
			log.Fatalf("Failed to connect to peer: %v", err)
		}
		fmt.Println("✅ Connected to peer successfully!")
	}


	// create DHT
	dht, err := kad.New(ctx, h)
	if err != nil {
		log.Fatalf("Failed to create DHT: %v", err)
	}

	if err := dht.Bootstrap(ctx); err != nil {
		log.Printf("DHT bootstrap warning: %v", err)
	}

	// periodically print connected peers
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			peers := h.Peerstore().Peers()
			fmt.Printf("--- Peerstore peers (%d) ---\n", len(peers))
			for _, p := range peers {
				if p == h.ID() {
					continue
				}
				fmt.Printf(" - %s\n", p.String())
			}
			if kadRouting, ok := interface{}(dht).(routing.Routing); ok {
				_ = kadRouting
			}
		}
	}()

	// start mDNS service (new API: returns one value, no context arg)
	svc := mdns.NewMdnsService(h, "gotangle-mdns", &mdnsNotifee{h: h})
	defer svc.Close()

	// wait for interrupt
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	fmt.Println("Shutting down...")
	ticker.Stop()
	_ = dht.Close()
	_ = h.Close()
}


