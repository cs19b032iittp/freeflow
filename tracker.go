package main

import (
	"context"
	"fmt"
	"log"
	"time"

	gorpc "github.com/libp2p/go-libp2p-gorpc"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

type Peer struct {
	addr string
}

type MetaChunk struct {
	Index      int64
	Hash       []byte
	Peers      []Peer
	NumOfPeers int64
}

type MetaTracker struct {
	FileHash    []byte
	FileInfo    string
	FileSize    int64
	ChunkSize   int64
	NumOfChunks int64
	Chunks      []MetaChunk
}

type TrackerService struct {
	trackers map[string]MetaTracker
}

type AnnounceChunksArgs struct {
	FileHash string
	Indexes  int64
	PeerAddr string
}

type AnnounceChunksReply struct{}

func (ts *TrackerService) AnnounceChunks(ctx context.Context, args AnnounceChunksArgs, reply *AnnounceChunksReply) error {
	fmt.Println("Got a AnnounceChunks RPC from peer: ", args.PeerAddr)
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

	// fmt.Printf("/ip4/127.0.0.1/tcp/%v/p2p/%s\n", cfg.listenPort, h.ID().Pretty())

	for {
		time.Sleep(time.Second * 1)
	}
}
