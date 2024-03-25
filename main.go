package main

import (
	"eigertest/filediff"
	"fmt"
)

func main() {
	originalFilePath := "phrases.txt"
	updatedFilePath := "updatedPhrases.txt"
	outputFilePath := "outputFilePhrases.txt"

	delta, err := filediff.GenerateDelta(originalFilePath, updatedFilePath)
	if err != nil {
		fmt.Printf("Error generating delta: %v\n", err)
		return
	}

	if err := filediff.ApplyDelta(originalFilePath, delta, outputFilePath); err != nil {
		fmt.Println("Error applying delta:", err)
	}
}
