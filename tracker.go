package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	gorpc "github.com/libp2p/go-libp2p-gorpc"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

// type Peer struct {
// 	ID peer.ID
// }

type MetaChunk struct {
	Index      int64
	Hash       string
	Peers      []peer.ID
	NumOfPeers int64
}

type MetaTracker struct {
	FileHash    string
	FileInfo    string
	FileSize    int64
	ChunkSize   int64
	NumOfChunks int64
	Chunks      []MetaChunk
}

var Trackers map[string]MetaTracker

type TrackerService struct {
}

type AnnounceChunksArgs struct {
	FileHash string
	Indexes  int64
	PeerAddr string
}

type AnnounceChunksReply struct{}

type AnnounceFileParams struct {
	M  MetaTracker
	ID peer.ID
}

type AnnounceFileResponse struct {
	Success bool
	Message string
}

type GetPeersParams struct {
	FileHash string
}

type GetPeersResponse struct {
	Chunks []MetaChunk
}

type GetChunkParams struct {
	FileHash string
}

type GetChunkResponse struct {
	Chunks []MetaChunk
}

func (ts *TrackerService) GetChunk(ctx context.Context, params GetPeersParams, reply *GetPeersResponse) error {
	fmt.Println("Got a GetPeers RPC")
	fmt.Println("lenth of trackers: ", len(Trackers))

	fmt.Println(Trackers[params.FileHash].Chunks)

	reply.Chunks = Trackers[params.FileHash].Chunks

	return nil
}

func (ts *TrackerService) GetPeers(ctx context.Context, params GetPeersParams, reply *GetPeersResponse) error {
	fmt.Println("Got a GetPeers RPC")
	fmt.Println("lenth of trackers: ", len(Trackers))

	fmt.Println(Trackers[params.FileHash].Chunks)

	reply.Chunks = Trackers[params.FileHash].Chunks

	return nil
}

func (ts *TrackerService) AnnounceChunks(ctx context.Context, args AnnounceChunksArgs, reply *AnnounceChunksReply) error {
	fmt.Println("Got a AnnounceChunks RPC from peer: ", args.PeerAddr)
	return nil
}

func (ts *TrackerService) AnnounceFile(ctx context.Context, params AnnounceFileParams, resp *AnnounceFileResponse) error {
	resp.Success = false
	m := params.M
	_, ok := Trackers[params.M.FileHash]
	if !ok {
		var wg sync.WaitGroup
		wg.Add(int(m.NumOfChunks))
		for i := int64(0); i < m.NumOfChunks; i++ {
			go func(chunkIndex int64) {
				defer wg.Done()

				m.Chunks[chunkIndex].NumOfPeers = 1

				peers := make([]peer.ID, 0)
				peers = append(peers, params.ID)
				m.Chunks[chunkIndex].Peers = peers
			}(i)
		}
		wg.Wait()

		Trackers[m.FileHash] = m

		resp.Success = true
		resp.Message = "File was open to share"
	} else {
		resp.Message = "File already exists"
	}

	fmt.Println(Trackers, len(Trackers))

	return nil
}

var trackerProtocol = protocol.ID("/p2p/rpc/tracker")

func CreateTracker(cfg *config) {
	h, err := makeHost(cfg.listenHost, cfg.listenPort)
	if err != nil {
		panic(err)
	}

	if err := setupDiscovery(h, "tracker"); err != nil {
		panic(err)
	}

	ctx := context.Background()

	// create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	topic, err := ps.Join("tracker")
	if err != nil {
		panic(err)
	}

	// and subscribe to it
	topic.Subscribe()

	rpcHost := gorpc.NewServer(h, trackerProtocol)

	Trackers = make(map[string]MetaTracker)
	svc := TrackerService{}
	err = rpcHost.Register(&svc)
	if err != nil {
		panic(err)
	}
	for _, addr := range h.Addrs() {
		ipfsAddr, err := multiaddr.NewMultiaddr("/ipfs/" + h.ID().Pretty())
		if err != nil {
			panic(err)
		}
		peerAddr := addr.Encapsulate(ipfsAddr)
		log.Printf("I'm listening on %s\n", peerAddr)
	}
	select {}
}
