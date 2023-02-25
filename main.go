package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"

	"github.com/multiformats/go-multiaddr"
)

var trackers = make(map[string]Meta)

func makeHost(host string, port int) (host.Host, error) {
	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", host, port))

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

	h, err := makeHost(cfg.listenHost, cfg.listenPort)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	// setup local mDNS discovery
	if err := setupDiscovery(h); err != nil {
		panic(err)
	}

	for {

		var option string
		fmt.Println("\x1b[32mChoose an option,")
		fmt.Println("1. Upload a file")
		fmt.Println("2. Download a file")
		fmt.Println("3. Shutdown\x1b[0m")
		fmt.Print("> ")
		fmt.Scanln(&option)

		switch option {
		case "1":
			var filePath string
			fmt.Println("\x1b[32mEnter the path of the file u want to seed,\x1b[0m")
			fmt.Print("> ")
			fmt.Scanln(&filePath)

			fileHash := GenerateMeta(filePath)
			fmt.Println(hex.EncodeToString(fileHash[:]))

			topicName := hex.EncodeToString(fileHash[:])
			topic, err := ps.Join(topicName)
			if err != nil {
				panic(err)
			}

			// and subscribe to it
			topic.Subscribe()

		case "2":
			var filePath string
			fmt.Println("\x1b[32mEnter the path of the meta file,\x1b[0m")
			fmt.Print("> ")
			fmt.Scanln(&filePath)

			m := &Meta{}
			DecodeMeta(filePath, m)
			fmt.Println(ps.ListPeers(hex.EncodeToString(m.Hash)))

		case "3":
			os.Exit(0)
		case "0":
			var peerId string
			fmt.Print("> ")
			fmt.Scanln(&peerId)

			Id, err := peer.Decode(peerId)
			if err != nil {
				panic(err)
			}

			// Get the peer's multiaddress
			addrs := h.Peerstore().Addrs(Id)
			if len(addrs) == 0 {
				panic(fmt.Errorf("No known addresses for peer %s", Id.Pretty()))
			}
			// addrs := h.Peerstore().Addrs(Id)
			for _, addr := range addrs {
				fmt.Printf("Connected to peer at %s\n", addr)
			}

		}

	}

}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", pi.ID.Pretty())
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID.Pretty(), err)
	}
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(h host.Host) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, "freeflow", &discoveryNotifee{h: h})
	return s.Start()
}
