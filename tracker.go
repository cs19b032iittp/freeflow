package main

import (
	"context"
	"fmt"
	"sync"

	gorpc "github.com/libp2p/go-libp2p-gorpc"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// type Peer struct {
// 	ID peer.ID
// }

type MetaPeer struct {
	ID     peer.ID
	Chunks []int64
}

type MetaTracker struct {
	FileHash string
	FileName string
	Peers    []MetaPeer
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
	FileHash    string
	FileName    string
	NumOfChunks int64
	ID          peer.ID
}

type AnnounceFileResponse struct {
	Success bool
	Message string
}

type GetPeersParams struct {
	FileHash string
}

type GetPeersResponse struct {
	Peers []MetaPeer
}

func (ts *TrackerService) GetPeers(ctx context.Context, params GetPeersParams, resp *GetPeersResponse) error {
	fmt.Println("Got a GetPeers RPC")
	fmt.Println("lenth of trackers: ", len(Trackers))

	fmt.Println(Trackers[params.FileHash].Peers)

	resp.Peers = Trackers[params.FileHash].Peers

	return nil
}

func (ts *TrackerService) AnnounceChunks(ctx context.Context, args AnnounceChunksArgs, reply *AnnounceChunksReply) error {
	fmt.Println("Got a AnnounceChunks RPC from peer: ", args.PeerAddr)
	return nil
}

func (ts *TrackerService) AnnounceFile(ctx context.Context, params AnnounceFileParams, resp *AnnounceFileResponse) error {
	fmt.Println("Got a AnnounceFile RPC from peer: ", params.ID)
	resp.Success = false
	_, ok := Trackers[params.FileHash]
	if !ok {

		metaPeers := make([]MetaPeer, 0)
		metaChunks := make([]int64, params.NumOfChunks)

		var wg sync.WaitGroup
		wg.Add(int(params.NumOfChunks))
		for i := int64(0); i < params.NumOfChunks; i++ {
			go func(chunkIndex int64) {
				defer wg.Done()
				metaChunks[chunkIndex] = 1
			}(i)
		}
		wg.Wait()

		metaPeers = append(metaPeers, MetaPeer{ID: params.ID, Chunks: metaChunks})
		Trackers[params.FileHash] = MetaTracker{FileHash: params.FileHash, FileName: params.FileName, Peers: metaPeers}

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
	select {}
}
