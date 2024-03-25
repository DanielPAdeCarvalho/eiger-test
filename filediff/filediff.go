package filediff

import (
	"bufio"
	"bytes"
	"eigertest/rollinghash"
	"fmt"
	"io"
	"os"
)

const (
	blockSize = 1024 // Define the size of the block to read from files, used for hashing and diff operations.
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
			_, _ = currentWindow.ReadByte()
		}

		if currentWindow.Len() == blockSize {
			rh := rollinghash.New(blockSize)
			tempWindow := currentWindow.Bytes()
			for _, b := range tempWindow {
				rh.AddByte(b)
			}

			hash := rh.GetHash()
			if indexes, exists := originalHashes[hash]; exists {
				deltaPosition := position + 1 - blockSize
				delta = append(delta, DeltaCommand{
					Command:    "copy",
					BlockIndex: indexes[0],
					Position:   deltaPosition,
				})
				currentWindow.Reset()
				position++
				continue
			}
		}
		position++
	}

	return delta, nil
}
