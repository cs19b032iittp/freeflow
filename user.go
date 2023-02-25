package main

import (
	"context"
	"fmt"

	gorpc "github.com/libp2p/go-libp2p-gorpc"

	"github.com/libp2p/go-libp2p/core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
)

func CreateUser(cfg *config) {
	h, err := makeHost(cfg.listenHost, cfg.listenPort)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// setup local mDNS discovery
	if err := setupDiscovery(h, "user"); err != nil {
		panic(err)
	}

	var server string
	fmt.Print("> ")
	fmt.Scanln(&server)

	ma, err := multiaddr.NewMultiaddr(server)
	if err != nil {
		panic(err)
	}

	fmt.Println("addrs: ", ma)
	peerInfo, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		panic(err)
	}

	fmt.Println("peerInfo: ", peerInfo)

	err = h.Connect(ctx, *peerInfo)
	if err != nil {
		panic(err)
	}

	fmt.Println("Connected")

	fmt.Println("PeerStore: ", h.Peerstore().PeerInfo(h.ID()))
	var args = AnnounceChunksArgs{PeerAddr: server}
	rpcClient := gorpc.NewClient(h, trackerProtocol)

	var reply AnnounceChunksReply
	err = rpcClient.Call(peerInfo.ID, "TrackerService", "AnnounceChunks", args, &reply)
	if err != nil {
		panic(err)
	}

}

/*
func CreateUser(cfg *config) {
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
	if err := setupDiscovery(h, "user"); err != nil {
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

*/
