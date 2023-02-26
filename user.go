package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"

	gorpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/libp2p/go-libp2p/core/peer"
)

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
	if err := setupDiscovery(h, "tracker"); err != nil {
		panic(err)
	}

	Files = make(map[string]Meta)
	svc := FileService{}
	rpcHost := gorpc.NewServer(h, fileProtocol)
	err = rpcHost.Register(&svc)
	if err != nil {
		panic(err)
	}

	rpcTracker := gorpc.NewClient(h, trackerProtocol)
	// rpcFile := gorpc.NewClient(h, fileProtocol)

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

			m := Meta{}
			fileHash := GenerateMeta(filePath, &m)
			fmt.Println(fileHash)

			trackers := ps.ListPeers("tracker")

			if len(trackers) == 0 {
				fmt.Println("Unable to connect to a tracker, Please try Again!")
				return
			}

			tracker := trackers[0]

			params := AnnounceFileParams{FileHash: m.Hash, ID: h.ID(), FileName: m.Name, NumOfChunks: m.NumOfChunks}
			resp := AnnounceFileResponse{}

			err = rpcTracker.Call(tracker, "TrackerService", "AnnounceFile", params, &resp)
			if err != nil {
				panic(err)
			}

			var wg sync.WaitGroup
			wg.Add(int(m.NumOfChunks))
			for i := int64(0); i < m.NumOfChunks; i++ {
				go func(chunkIndex int64) {
					defer wg.Done()
					m.Chunks[chunkIndex].Downloaded = true
				}(i)
			}
			wg.Wait()

			Files[m.Hash] = m
			fmt.Println(resp)

		case "2":
			var filePath string
			fmt.Println("\x1b[32mEnter the path of the meta file,\x1b[0m")
			fmt.Print("> ")
			fmt.Scanln(&filePath)

			m := &Meta{}
			DecodeMeta(filePath, m)

			trackers := ps.ListPeers("tracker")

			if len(trackers) == 0 {
				fmt.Println("Unable to connect to a tracker, Please try Again!")
				return
			}

			tracker := trackers[0]

			peers := make([]MetaPeer, 0)
			var resp = GetPeersResponse{Peers: peers}
			var params = GetPeersParams{FileHash: m.Hash}
			fmt.Println(m.Hash)
			err = rpcTracker.Call(tracker, "TrackerService", "GetPeers", params, &resp)
			if err != nil {
				panic(err)
			}

			fmt.Println(resp)

			// for {

			// 	peers := resp.Peers
			// 	peerList := []peer.ID
			// 	tmp_map = make(map[int64])

			// }

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


	// for {
	// 	var server string

	// 	fmt.Print("> ")
	// 	fmt.Scanln(&server)

	// 	ma, err := multiaddr.NewMultiaddr(server)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	fmt.Println("addrs: ", ma)
	// 	peerInfo, err := peer.AddrInfoFromP2pAddr(ma)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	fmt.Println("peerInfo: ", peerInfo)

	// 	err = h.Connect(ctx, *peerInfo)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	fmt.Println("Connected")

	// 	fmt.Println("PeerStore: ", h.Peerstore().PeerInfo(h.ID()))
	// 	var args = AnnounceChunksArgs{PeerAddr: server}
	// 	rpcClient := gorpc.NewClient(h, trackerProtocol)

	// 	var reply AnnounceChunksReply
	// 	err = rpcClient.Call(peerInfo.ID, "TrackerService", "AnnounceChunks", args, &reply)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }



}

*/
