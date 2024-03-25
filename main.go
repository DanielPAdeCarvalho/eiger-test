package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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

// Updates the hash with a block of data instead of byte by byte.
func (rh *RollingHash) HashData(data []byte) {
	rh.hash = 0 // Reset hash for new data block
	for i, b := range data {
		if i == 0 {
			rh.first = int(b) // Record the first byte for rolling
		}
		rh.AddByte(b)
	}
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

// Separated filesystem operations from hashing logic
func hashFileBlocks(filePath string) (map[int][]int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hashes := make(map[int][]int)
	reader := bufio.NewReader(file)
	buffer := make([]byte, blockSize)
	index := 0

	for {
		bytesRead, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if bytesRead == 0 {
			break
		}

		rh := NewRollingHash(bytesRead)
		rh.HashData(buffer[:bytesRead])

		hash := rh.GetHash()
		if _, exists := hashes[hash]; !exists {
			hashes[hash] = make([]int, 0)
		}
		hashes[hash] = append(hashes[hash], index)

		index++
	}

	return hashes, nil
}

func applyDelta(originalFilePath, deltaFilePath, outputFilePath string) error {
	// Open the original and delta files
	originalFile, err := os.Open(originalFilePath)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	deltaFile, err := os.Open(deltaFilePath)
	if err != nil {
		return err
	}
	defer deltaFile.Close()

	// Create the output file
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Read and apply each command from the delta file
	scanner := bufio.NewScanner(deltaFile)
	for scanner.Scan() {
		command := scanner.Text()
		if strings.HasPrefix(command, "copy block") {
			// Parse the command to get block index and position
			parts := strings.Split(command, " ")
			blockIndex, err := strconv.Atoi(parts[2])
			if err != nil {
				return err
			}
			position, err := strconv.Atoi(parts[5])
			if err != nil {
				return err
			}

			// Calculate the offset in the original file and seek to it
			offset := int64(blockIndex * blockSize)
			_, err = originalFile.Seek(offset, io.SeekStart)
			if err != nil {
				return err
			}

			// Copy the block from the original file to the output file
			_, err = io.CopyN(outputFile, originalFile, blockSize)
			if err != nil && err != io.EOF {
				return err
			}

			// Move the write position of the output file
			_, err = outputFile.Seek(int64(position), io.SeekStart)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(command, "insert at position") {
			// Parse the command to get position and data
			insertCmdParts := strings.SplitN(command, ": ", 2)
			positionPart := strings.Split(insertCmdParts[0], " ")
			position, err := strconv.Atoi(positionPart[3])
			if err != nil {
				return err
			}

			data, err := hex.DecodeString(insertCmdParts[1])
			if err != nil {
				return err
			}

			// Move the write position of the output file and insert the data
			_, err = outputFile.Seek(int64(position), io.SeekStart)
			if err != nil {
				return err
			}
			_, err = outputFile.Write(data)
			if err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func generateDelta(originalFilePath, updatedFilePath string) ([]string, error) {
	originalHashes, err := hashFileBlocks(originalFilePath)
	if err != nil {
		return nil, err
	}

	updatedFile, err := os.Open(updatedFilePath)
	if err != nil {
		return nil, err
	}
	defer updatedFile.Close()

	reader := bufio.NewReader(updatedFile)
	var delta []string
	var currentWindow bytes.Buffer
	position := 0

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				// Handle any remaining bytes in the window as inserts
				if currentWindow.Len() > 0 {
					hexData := hex.EncodeToString(currentWindow.Bytes())
					delta = append(delta, fmt.Sprintf("insert at position %d: %s", position-currentWindow.Len(), hexData))
				}
				break
			}
			return nil, err
		}

		currentWindow.WriteByte(b)
		if currentWindow.Len() > blockSize {
			_, _ = currentWindow.ReadByte() // Discard the oldest byte to maintain window size
		}

		// Generate hash for the current window and compare with original file hashes
		if currentWindow.Len() == blockSize {
			rh := NewRollingHash(blockSize)
			tempWindow := currentWindow.Bytes()
			for _, b := range tempWindow {
				rh.AddByte(b)
			}

			hash := rh.GetHash()
			if indexes, exists := originalHashes[hash]; exists {
				// Confirm the block truly matches to handle hash collisions
				delta = append(delta, fmt.Sprintf("copy block %d to position %d", indexes[0], position+1-blockSize))
				currentWindow.Reset()
				continue
			}
		}

		position++
	}

	return delta, nil
}

func main() {
	originalFilePath := "phrases.txt"
	updatedFilePath := "updatedPhrases.txt"
	outputFilePath := "outputFilePhrases.txt"

	delta, err := generateDelta(originalFilePath, updatedFilePath)
	if err != nil {
		fmt.Printf("Error generating delta: %v\n", err)
		return
	}

	fmt.Println("Delta commands:")
	for _, cmd := range delta {
		fmt.Println(cmd)
	}

	if err := applyDelta(originalFilePath, "path/to/delta/commands", outputFilePath); err != nil {
		fmt.Println("Error applying delta:", err)
	}

}
