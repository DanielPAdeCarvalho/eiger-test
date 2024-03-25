package main

import (
	"bufio"
	"bytes"
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
type DeltaCommand struct {
	Command    string
	Position   int    // Position to insert or destination position for copy
	BlockIndex int    // Used for copy commands to indicate the block index in the original file
	Data       []byte // Used for insert commands to hold the binary data
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

func applyDelta(originalFilePath string, deltaCommands []DeltaCommand, outputFilePath string) error {
	originalFile, err := os.Open(originalFilePath)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	outputFile, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	for i, command := range deltaCommands {
		fmt.Printf("Executing command #%d: %s\n", i, command.Command)
		switch command.Command {
		case "copy":
			offset := int64(command.BlockIndex * blockSize)
			fmt.Printf("Copy: Seeking to %d in original file (BlockIndex: %d)\n", offset, command.BlockIndex)

			_, err = originalFile.Seek(offset, io.SeekStart)
			if err != nil {
				return err
			}

			outputFileSeek, err := outputFile.Seek(int64(command.Position), io.SeekStart)
			if err != nil {
				return err
			}
			fmt.Printf("Copy: Seeking to %d in output file (Command Position: %d)\n", outputFileSeek, command.Position)

			copiedBytes, err := io.CopyN(outputFile, originalFile, blockSize)
			if err != nil && err != io.EOF {
				return err
			}
			fmt.Printf("Copy: Copied %d bytes from original to output\n", copiedBytes)

		case "insert":
			outputFileSeek, err := outputFile.Seek(int64(command.Position), io.SeekStart)
			if err != nil {
				return err
			}
			fmt.Printf("Insert: Seeking to %d in output file (Command Position: %d)\n", outputFileSeek, command.Position)

			writtenBytes, err := outputFile.Write(command.Data)
			if err != nil {
				return err
			}
			fmt.Printf("Insert: Wrote %d bytes to output file\n", writtenBytes)

		default:
			return fmt.Errorf("unknown command: %s", command.Command)
		}
	}

	return nil
}

func generateDelta(originalFilePath, updatedFilePath string) ([]DeltaCommand, error) {
	originalHashes, err := hashFileBlocks(originalFilePath)
	if err != nil {
		fmt.Println("Error hashing original file blocks:", err)
		return nil, err
	}

	updatedFile, err := os.Open(updatedFilePath)
	if err != nil {
		fmt.Println("Error opening updated file:", err)
		return nil, err
	}
	defer updatedFile.Close()

	reader := bufio.NewReader(updatedFile)
	var delta []DeltaCommand
	var currentWindow bytes.Buffer
	position := 0

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				// Handle any remaining bytes in the window as inserts
				if currentWindow.Len() > 0 {
					fmt.Printf("Insert command at EOF - Position: %d, Data Length: %d\n", position-currentWindow.Len(), currentWindow.Len())
					delta = append(delta, DeltaCommand{
						Command:  "insert",
						Position: position - currentWindow.Len(),
						Data:     currentWindow.Bytes(),
					})
				}
				break
			}
			fmt.Println("Error reading byte from updated file:", err)
			return nil, err
		}

		currentWindow.WriteByte(b)
		if currentWindow.Len() > blockSize {
			// This log helps understand when and why the oldest byte is discarded
			fmt.Printf("Discarding oldest byte to maintain window size - Current Position: %d\n", position)
			_, _ = currentWindow.ReadByte()
		}

		if currentWindow.Len() == blockSize {
			rh := NewRollingHash(blockSize)
			tempWindow := currentWindow.Bytes()
			for _, b := range tempWindow {
				rh.AddByte(b)
			}

			hash := rh.GetHash()
			if indexes, exists := originalHashes[hash]; exists {
				deltaPosition := position + 1 - blockSize
				// Log the creation of a copy command, including the positions and block index involved
				fmt.Printf("Copy command - Block Index: %d, Position: %d\n", indexes[0], deltaPosition)
				delta = append(delta, DeltaCommand{
					Command:    "copy",
					BlockIndex: indexes[0],
					Position:   deltaPosition,
				})
				currentWindow.Reset()
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

	fmt.Println("Delta commands to be executed:")
	for _, cmd := range delta {
		fmt.Println(cmd)
	}

	if err := applyDelta(originalFilePath, delta, outputFilePath); err != nil {
		fmt.Println("Error applying delta:", err)
	}
}
