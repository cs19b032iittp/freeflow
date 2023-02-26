package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"

	"github.com/multiformats/go-multiaddr"
)

func makeHost(host string, port string) (host.Host, error) {
	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", host, port))

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	return libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
}

func main() {
	log.Println("Application Started,")

	cfg := parseFlags()
	fmt.Println("flags: ", cfg)

	switch cfg.role {
	case "tracker":
		CreateTracker(cfg)
	case "user":
		CreateUser(cfg)
	}

}

type discoveryNotifee struct {
	h host.Host
	r string
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID.Pretty(), err)
	}
}

func setupDiscovery(h host.Host, role string) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, role, &discoveryNotifee{h: h, r: role})
	return s.Start()
}
