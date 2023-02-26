package main

import (
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/protocol"
)

const chunkSize = 128

type Chunk struct {
	Index      int64
	Hash       string
	Downloaded bool
}

type Meta struct {
	Name        string
	FileSize    int64
	ChunkSize   int64
	NumOfChunks int64
	Hash        string
	Chunks      []Chunk
	FilePath    string
}

var Files map[string]Meta

type FileService struct {
}

type GetChunkParams struct {
	FileHash   string
	ChunkIndex int64
	Size       int64
	Offset     int64
}

type GetChunkResponse struct {
	Data []byte
	Size int64
}

var fileProtocol = protocol.ID("/p2p/rpc/file")

func (fs *FileService) GetChunk(ctx context.Context, params GetChunkParams, resp *GetChunkResponse) error {
	fmt.Println("Got a GetChunk RPC")
	meta := Files[params.FileHash]
	fmt.Println(meta.FilePath)
	inputFile, err := os.Open(meta.FilePath)
	if err != nil {
		panic(err)
	}
	defer inputFile.Close()

	buffer := make([]byte, params.Size)
	_, err = inputFile.ReadAt(buffer, params.Offset)
	if err != nil {
		panic(err)
	}

	resp.Data = buffer

	return nil
}

// Hashes the given data and returns its SHA256 hash.
func hashContent(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func GenerateMeta(filepath string, m *Meta) string {

	inputFile, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer inputFile.Close()

	fileInfo, err := inputFile.Stat()
	if err != nil {
		panic(err)
	}

	m.FileSize = fileInfo.Size()
	m.ChunkSize = chunkSize

	m.NumOfChunks = m.FileSize / chunkSize
	if m.FileSize%chunkSize != 0 {
		m.NumOfChunks++
	}

	fileData := make([]byte, m.FileSize)
	inputFile.Read(fileData)

	m.Hash = hashContent(fileData)
	m.Name = fileInfo.Name()

	var wg sync.WaitGroup
	wg.Add(int(m.NumOfChunks))

	m.Chunks = make([]Chunk, m.NumOfChunks)

	fmt.Println(m.NumOfChunks)
	for i := int64(0); i < m.NumOfChunks; i++ {
		go func(chunkIndex int64) {
			defer wg.Done()
			offset := chunkIndex * chunkSize
			chunkSizeActual := chunkSize
			if chunkIndex == m.NumOfChunks-1 {
				chunkSizeActual = int(m.FileSize - offset)
			}

			chunk := make([]byte, chunkSizeActual)
			_, err := inputFile.ReadAt(chunk, offset)
			if err != nil {
				panic(err)
			}

			c := Chunk{}
			c.Index = chunkIndex
			c.Hash = hashContent(chunk)
			m.Chunks[chunkIndex] = c
		}(i)
	}

	wg.Wait()

	substrings := strings.Split(m.Name, ".")

	metaFile, err := os.Create("./metas/" + substrings[len(substrings)-2] + ".meta")
	if err != nil {
		log.Fatal(err)
	}
	encoder := gob.NewEncoder(metaFile)
	err = encoder.Encode(m)
	if err != nil {
		log.Fatal(err)
	}
	metaFile.Close()
	return m.Hash
}

func DecodeMeta(filepath string, m *Meta) {

	metaFile, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}

	decoder := gob.NewDecoder(metaFile)
	err = decoder.Decode(m)
	if err != nil {
		log.Fatal(err)
	}

	metaFile.Close()

}
