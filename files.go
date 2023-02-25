package main

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

const chunkSize = 128

type Chunk struct {
	Index      int64
	Hash       []byte
	Downloaded bool
}

type Meta struct {
	Name        string
	FileSize    int64
	ChunkSize   int64
	NumOfChunks int64
	Hash        []byte
	Chunks      []Chunk
}

// Hashes the given data and returns its SHA256 hash.
func hashContent(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

func GenerateMeta(filepath string) []byte {
	m := &Meta{}

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

	m.Chunks = []Chunk{}

	fmt.Println(m.NumOfChunks)
	mu := sync.Mutex{}
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
			c.Downloaded = true

			mu.Lock()
			m.Chunks = append(m.Chunks, c)
			mu.Unlock()

			// Do something with the chunk
			fmt.Printf("Chunk %d: %s\n", chunkIndex, string(chunk))
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

	trackers[hex.EncodeToString(m.Hash)] = *m

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

	var wg sync.WaitGroup
	wg.Add(int(m.NumOfChunks))

	for i := int64(0); i < m.NumOfChunks; i++ {
		go func(chunkIndex int64) {
			defer wg.Done()
			m.Chunks[chunkIndex].Downloaded = false
		}(i)
	}

	wg.Wait()

	metaFile.Close()

}
