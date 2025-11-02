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
	peerstore "github.com/libp2p/go-libp2p/core/peer"
	routing "github.com/libp2p/go-libp2p/core/routing"
	kad "github.com/libp2p/go-libp2p-kad-dht"
	discovery "github.com/libp2p/go-libp2p/core/discovery"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// public key= peer ID
// private key = Your secret, used for encryption

type mdnsNotifee struct{
	h host.Host
	// it stores node h , so we can connect to new peers when found
}

func (n *mdnsNotifee) HandlePeerFound(pi peerstore.AddrInfo) {
	fmt.Printf("mDNS discovered: %s\n", pi.ID.Pretty())
	if err := n.h.Connect(context.Background(), pi); err != nil {
		fmt.Printf("Error connecting to discovered peer %s: %s\n", pi.ID.Pretty(), err)
	}
}

// tcp = listening on 0.0.0.0:9000 (TCP)
//  Other node can connect via: /ip4/192.168.1.10/tcp/9000/p2p/<peerID>


// mDNS = Multicast DNS — a way for devices on the same local network 
// (like your Wi-Fi) to find each other automatically, without typing IPs.

//If another machine runs the same program, it replies:
//“Yes, I’m here — here’s my Peer ID and address.”

func main() {
	port := flag.Int("port", 0, "port to listen on (0 = random)")
	flag.Parse()

	ctx := context.Background()

	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}
	
	// build the p2p node (the host)
	/// Assign it your identity (private key)
	// Make it listen on a TCP port (For incoming connections)
	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port)),
	)
	if err != nil {
		log.Fatalf("Failed to create libp2p host: %v", err)
	}

	fmt.Printf("Hello! I am a libp2p node. Peer ID: %s\n", h.ID().Pretty())
	for _, a := range h.Addrs() {
		fmt.Printf(" Listening on: %s/p2p/%s\n", a.String(), h.ID().Pretty())
	}

	// created a distributed hash table
	// this is how peer store and find each other in large decentralized neetworks
	dht, err := kad.New(ctx, h)
	if err != nil {
		log.Fatalf("Failed to create DHT: %v", err)
	}

	
	// connceting to know peers if it exist
	if err := dht.Bootstrap(ctx); err != nil {
		log.Printf("DHT bootstrap warning: %v", err)
	}

	// every 15s it prints who is in your peer list
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			peers := h.Peerstore().Peers()
			fmt.Printf("--- Peerstore peers (%d) ---\n", len(peers))
			for _, p := range peers {
				if p == h.ID() { continue }
				fmt.Printf(" - %s\n", p.Pretty())
			}

			
			if kadRouting, ok := interface{}(dht).(routing.Routing); ok {
				_ = kadRouting 
			}
		}
	}()

	// this makes your node broadcast
	svc := mdns.NewMdnsService(h, "gotangle-mdns", &mdnsNotifee{h: h})
	if svc == nil {
		log.Printf("mDNS service nil (couldn't start mDNS)")
	} else {
		defer svc.Close()
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	fmt.Println("Shutting down...")
	ticker.Stop()
	_ = h.Close()
}
