package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

const (
	base      = 256       // Base value to use for hashing; 256 works well for binary files and text.
	modPrime  = 805306457 // A large prime number for modulo operation to avoid hash collisions.
	blockSize = 1024      // Block size to use for reading files.
)

// RollingHash struct to keep track of the hash and the factors for rolling.
type RollingHash struct {
	hash   int // Current hash value
	window int // Size of the window (number of bytes)
	first  int // The value of the first byte in the window
	baseN  int // base raised to the power of (window-1), modulo modPrime
}

// NewRollingHash initializes a new RollingHash for a given window size.
func NewRollingHash(window int) *RollingHash {
	rh := &RollingHash{window: window}
	rh.baseN = 1
	for i := 0; i < window-1; i++ {
		rh.baseN = (rh.baseN * base) % modPrime
	}
	return rh
}

// AddByte updates the hash with a new byte, adding it to the calculation.
func (rh *RollingHash) AddByte(b byte) {
	rh.hash = (rh.hash*base + int(b)) % modPrime
}

// RemoveByte removes the first byte from the hash calculation.
func (rh *RollingHash) RemoveByte(b byte) {
	rh.hash = (rh.hash + modPrime - rh.first*rh.baseN%modPrime) % modPrime
	rh.first = int(b)
}

// Roll updates the hash to exclude the oldest byte and include a new byte.
func (rh *RollingHash) Roll(oldByte, newByte byte) {
	rh.RemoveByte(oldByte)
	rh.AddByte(newByte)
}

// GetHash returns the current hash value.
func (rh *RollingHash) GetHash() int {
	return rh.hash
}

func hashFileBlocks(filePath string) (map[int][]int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hashes := make(map[int][]int) // Map of hash values to block indexes.
	reader := bufio.NewReader(file)
	buffer := make([]byte, blockSize)
	index := 0

	for {
		bytesRead, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if bytesRead == 0 {
			break
		}

		// Initialize a new RollingHash for each block.
		rh := NewRollingHash(bytesRead) // Use bytesRead instead of blockSize for the last block which might be smaller.
		for _, b := range buffer[:bytesRead] {
			rh.AddByte(b)
		}

		hash := rh.GetHash()
		// Store the hash along with its block index. If collisions are expected, you can append index to a slice.
		if _, exists := hashes[hash]; !exists {
			hashes[hash] = make([]int, 0)
		}
		hashes[hash] = append(hashes[hash], index)

		index++
	}

	return hashes, nil
}

func main() {
	filePath := "phrases.txt" // Update with file path for file
	hashes, err := hashFileBlocks(filePath)
	if err != nil {
		fmt.Println("Error hashing file blocks:", err)
		return
	}

	for hash, indexes := range hashes {
		fmt.Printf("Hash: %d, Blocks: %+v\n", hash, indexes)
	}
}
