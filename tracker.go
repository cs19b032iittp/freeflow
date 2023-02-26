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

type MetaTracker struct {
	FileHash string
	FileName string
	Peers    map[peer.ID][]int64
}

var Trackers map[string]MetaTracker

type TrackerService struct {
}

type AnnounceChunksParams struct {
	FileHash string
	Chunks   []int64
	ID       peer.ID
}

type AnnounceChunksResponse struct{}

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
	Peers map[peer.ID][]int64
}

func (ts *TrackerService) GetPeers(ctx context.Context, params GetPeersParams, resp *GetPeersResponse) error {
	fmt.Println("Got a GetPeers RPC")
	fmt.Println(Trackers[params.FileHash].Peers)
	resp.Peers = Trackers[params.FileHash].Peers

	return nil
}

func (ts *TrackerService) AnnounceChunks(ctx context.Context, params AnnounceChunksParams, resp *AnnounceChunksResponse) error {
	fmt.Println("Got a AnnounceChunks RPC from peer: ", params.ID.Pretty())

	m, ok := Trackers[params.FileHash]

	var mutex sync.Mutex
	if ok {
		mutex.Lock()
		m.Peers[params.ID] = params.Chunks
		mutex.Unlock()
	}

	return nil
}

func (ts *TrackerService) AnnounceFile(ctx context.Context, params AnnounceFileParams, resp *AnnounceFileResponse) error {
	fmt.Println("Got a AnnounceFile RPC from peer: ", params.ID)
	resp.Success = false
	_, ok := Trackers[params.FileHash]
	if !ok {

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

		// metaPeers = append(metaPeers, MetaPeer{ID: params.ID, Chunks: metaChunks})
		Peers := make(map[peer.ID][]int64)
		Peers[params.ID] = metaChunks
		Trackers[params.FileHash] = MetaTracker{FileHash: params.FileHash, FileName: params.FileName, Peers: Peers}

		resp.Success = true
		resp.Message = "File was open to share"
	} else {
		resp.Message = "File already exists"
	}

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
