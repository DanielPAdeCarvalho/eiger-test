package main

import (
	"eigertest/filediff"
	"fmt"
	"os"
)

func main() {
	// Check if the correct number of command-line arguments are provided.
	// The program expects 3 arguments: originalFilePath, updatedFilePath, and outputFilePath.
	if len(os.Args) < 4 {
		// If not enough arguments are provided, print the usage instructions and exit.
		fmt.Println("Usage: <program> <originalFilePath> <updatedFilePath> <outputFilePath>")
		return
	}

	originalFilePath := os.Args[1] // The path to the original file.
	updatedFilePath := os.Args[2]  // The path to the updated file.
	outputFilePath := os.Args[3]   // The path where the output file will be saved.

	// Generate the delta between the original and updated files.
	// This delta describes the changes needed to transform the original file into the updated file.
	delta, err := filediff.GenerateDelta(originalFilePath, updatedFilePath)
	if err != nil {
		// If an error occurs during delta generation, print the error and exit.
		fmt.Printf("Error generating delta: %v\n", err)
		return
	}

	// Apply the generated delta to the original file to produce the output file.
	if err := filediff.ApplyDelta(originalFilePath, delta, outputFilePath); err != nil {
		fmt.Println("Error applying delta:", err)
	}
}
