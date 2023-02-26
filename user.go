package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

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
	rpcFile := gorpc.NewClient(h, fileProtocol)

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
			GenerateMeta(filePath, &m)

			trackers := ps.ListPeers("tracker")

			if len(trackers) == 0 {
				fmt.Println("Unable to connect to a tracker, Please try Again!")
				continue
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

			if resp.Success {
				fmt.Println(resp.Message)
			} else {
				fmt.Println(resp.Message)

			}

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
				continue
			}

			tracker := trackers[0]

			peers := make(map[peer.ID][]int64, 0)
			var resp = GetPeersResponse{Peers: peers}
			var params = GetPeersParams{FileHash: m.Hash}
			fmt.Println(m.Hash)
			err = rpcTracker.Call(tracker, "TrackerService", "GetPeers", params, &resp)
			if err != nil {
				panic(err)
			}

			if len(resp.Peers) == 0 {
				fmt.Println("Unable to connect to peers, Please try again after sometime!")
				continue
			}

			chunkMap := make(map[int][]peer.ID)

			for i := int64(0); i < m.NumOfChunks; i++ {
				peerlist := make([]peer.ID, 0)
				chunkMap[int(i)] = peerlist
			}

			for PeerID, ChunksList := range resp.Peers {
				for idx, val := range ChunksList {
					if val == 1 {
						chunkMap[idx] = append(chunkMap[idx], PeerID)
					}
				}
			}

			outputFile, err := os.Create(m.Name)
			if err != nil {
				panic(err)
			}

			// Set the file size
			fileSize := m.FileSize
			err = outputFile.Truncate(fileSize)
			if err != nil {
				panic(err)
			}

			var start = false

			var acWg sync.WaitGroup // Announce Chunk Wait Group
			acWg.Add(1)
			go func() {

				for {

					if start {

						Chunks := make([]int64, m.NumOfChunks)
						for i := int64(0); i < m.NumOfChunks; i++ {
							if m.Chunks[i].Downloaded {
								Chunks[i] = 1
							} else {
								Chunks[i] = 0
							}
						}

						params := AnnounceChunksParams{FileHash: m.Hash, ID: h.ID(), Chunks: Chunks}
						err = rpcTracker.Call(tracker, "TrackerService", "AnnounceChunks", params, &resp)
						if err != nil {
							panic(err)
						}
					}

					time.Sleep(time.Millisecond * 2000)
				}

			}()

			var wg sync.WaitGroup
			wg.Add(int(m.NumOfChunks))
			var mutex = &sync.Mutex{}
			for idx := int64(0); idx < m.NumOfChunks; idx++ {
				go func(i int64) {
					defer wg.Done()

					t := time.Second * time.Duration(rand.Intn(int(m.NumOfChunks-2)+2))
					time.Sleep(t)

					flag := true
					for flag {
						ID := chunkMap[int(i)][0]
						offset := i * m.ChunkSize
						size := m.ChunkSize

						if i == m.NumOfChunks-1 {
							size = m.FileSize - offset
						}
						params := GetChunkParams{FileHash: m.Hash, ChunkIndex: i, Size: size, Offset: offset}
						resp := GetChunkResponse{}

						err = rpcFile.Call(ID, "FileService", "GetChunk", params, &resp)
						if err == nil {

							hash1 := []byte(m.Chunks[int(i)].Hash)
							hash2 := []byte(hashContent(resp.Data))

							if subtle.ConstantTimeCompare(hash1[:], hash2[:]) == 1 {

								start = true

								mutex.Lock()
								_, err = outputFile.Seek(offset, 0)
								if err != nil {
									panic(err)
								}

								// Write the data to the file
								_, err := outputFile.Write(resp.Data)
								if err != nil {
									panic(err)
								}

								mutex.Unlock()

								m.Chunks[i].Downloaded = true
								flag = false
								fmt.Println("Downloaded Chunk: ", i)
							} else {
								flag = true
							}

						}

					}
				}(idx)
			}

			acWg.Done()
			wg.Wait()
			acWg.Wait()

			outputFile.Close()

			fmt.Println(m.Name, "was downloaded")

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
