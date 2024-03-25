package filediff

import (
	"bufio"
	"bytes"
	"eigertest/rollinghash"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
)

const (
	blockSize   = 1024     // Define the size of the block to read from files, used for hashing and diff operations.
	sectionSize = 10485760 // 10MB sections
)

// DeltaCommand defines a structure for delta commands indicating how to transform the original file into the updated version.
type DeltaCommand struct {
	Command    string
	Position   int
	BlockIndex int
	Data       []byte
}

// hashFileBlocks computes and returns a map of hash values to their corresponding block indices in the specified file.
// This function facilitates identifying unique blocks and their positions for generating deltas.
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

		rh := rollinghash.New(bytesRead)
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

// ApplyDelta applies a series of delta commands to transform the original file into its updated version, resulting in a new output file.
// It manages file seeking and writes based on the delta instructions, handling both copy and insert operations.
func ApplyDelta(originalFilePath string, deltaCommands []DeltaCommand, outputFilePath string) error {
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

	for _, command := range deltaCommands {
		switch command.Command {
		case "copy":
			offset := int64(command.BlockIndex * blockSize)
			_, err = originalFile.Seek(offset, io.SeekStart)
			if err != nil {
				return err
			}

			_, err := outputFile.Seek(int64(command.Position), io.SeekStart)
			if err != nil {
				return err
			}
			_, err = io.CopyN(outputFile, originalFile, blockSize)
			if err != nil && err != io.EOF {
				return err
			}

		case "insert":
			_, err = outputFile.Seek(int64(command.Position), io.SeekStart)
			if err != nil {
				return err
			}
			_, err = outputFile.Write(command.Data)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown command: %s", command.Command)
		}
	}

	return nil
}

// GenerateDelta analyzes the differences between an original and an updated file,
// producing a series of delta commands that describe how to transform the original file into the updated version.
// This function leverages rolling hashing to efficiently identify matching blocks and generate appropriate commands.
func GenerateDelta(originalFilePath, updatedFilePath string) ([]DeltaCommand, error) {
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

	fileInfo, err := updatedFile.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()
	numSections := int(fileSize) / sectionSize

	var wg sync.WaitGroup
	deltaChan := make(chan []DeltaCommand, numSections)

	for i := 0; i <= numSections; i++ {
		start := i * sectionSize
		end := start + sectionSize
		if end > int(fileSize) {
			end = int(fileSize)
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			sectionDelta, _ := processSection(originalHashes, updatedFilePath, start, end)
			deltaChan <- sectionDelta
		}(start, end)
	}

	go func() {
		wg.Wait()
		close(deltaChan)
	}()

	var deltas []DeltaCommand
	for sectionDelta := range deltaChan {
		deltas = append(deltas, sectionDelta...)
	}

	// Sort the deltas slice by Position to ensure they are in order
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].Position < deltas[j].Position
	})

	return deltas, nil
}

func processSection(originalHashes map[int][]int, filePath string, start, end int) ([]DeltaCommand, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Seek to the start position of this section
	_, err = file.Seek(int64(start), io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("error seeking file: %w", err)
	}

	reader := bufio.NewReader(file)
	var delta []DeltaCommand
	var currentWindow bytes.Buffer
	position := start

	// Adjust the read loop to stop when reaching the end of the section
	for position < end {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break // End of file is expected, depending on section end
			}
			return nil, fmt.Errorf("error reading byte from file: %w", err)
		}

		currentWindow.WriteByte(b)
		if currentWindow.Len() > blockSize {
			_, _ = currentWindow.ReadByte() // Keep the window size constant
		}

		// Ensure we only process full blocks or the last block in the section
		if currentWindow.Len() == blockSize || position == end-1 {
			rh := rollinghash.New(min(blockSize, currentWindow.Len()))
			tempWindow := currentWindow.Bytes()
			for _, b := range tempWindow {
				rh.AddByte(b)
			}

			hash := rh.GetHash()
			if indexes, exists := originalHashes[hash]; exists && len(indexes) > 0 {
				// Generate copy command if hash matches
				deltaPosition := max(position+1-blockSize, start)
				delta = append(delta, DeltaCommand{
					Command:    "copy",
					BlockIndex: indexes[0], // Assuming first occurrence is the match
					Position:   deltaPosition,
				})
				currentWindow.Reset() // Reset window after matching
			} else if position == end-1 {
				// Insert the remaining bytes at the end of the section
				deltaPosition := position - currentWindow.Len() + 1
				delta = append(delta, DeltaCommand{
					Command:  "insert",
					Position: deltaPosition,
					Data:     tempWindow,
				})
				currentWindow.Reset()
			}
		}

		position++
	}

	// Handle any remaining bytes in the window as inserts, if not already done
	if currentWindow.Len() > 0 {
		fmt.Printf("Insert command at section end - Position: %d, Data Length: %d\n", position-currentWindow.Len(), currentWindow.Len())
		delta = append(delta, DeltaCommand{
			Command:  "insert",
			Position: position - currentWindow.Len(),
			Data:     currentWindow.Bytes(),
		})
	}

	return delta, nil
}

// Helper function to ensure we don't exceed buffer bounds
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function to ensure we correctly position delta commands
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
