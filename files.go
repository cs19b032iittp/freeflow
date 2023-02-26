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
	Index int64
	Hash  string
}

type Meta struct {
	Name        string
	FileSize    int64
	ChunkSize   int64
	NumOfChunks int64
	Hash        string
	Chunks      []Chunk
}

// Hashes the given data and returns its SHA256 hash.
func hashContent(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func GenerateMeta(filepath string, m *MetaTracker) string {

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

	m.FileHash = hashContent(fileData)
	m.FileInfo = fileInfo.Name()

	var wg sync.WaitGroup
	wg.Add(int(m.NumOfChunks))

	m.Chunks = make([]MetaChunk, m.NumOfChunks)

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

			c := MetaChunk{}
			c.Index = chunkIndex
			c.Hash = hashContent(chunk)
			m.Chunks[chunkIndex] = c
		}(i)
	}

	wg.Wait()

	substrings := strings.Split(m.FileInfo, ".")

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
	return m.FileHash
}

func DecodeMeta(filepath string, m *MetaTracker) {

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
