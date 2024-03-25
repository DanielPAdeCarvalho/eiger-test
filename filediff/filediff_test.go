package filediff

import (
	"bytes"
	"os"
	"testing"
)

// TestHashFileBlocks tests the hashFileBlocks function for a basic case.
func TestHashFileBlocks(t *testing.T) {
	// Setup: create a temporary file with known content
	content := []byte("hello world")
	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Act: hash the file blocks
	hashes, err := hashFileBlocks(tmpfile.Name())
	if err != nil {
		t.Errorf("hashFileBlocks returned error: %v", err)
	}

	// Assert: Check if the hashes map is not empty (basic check)
	if len(hashes) == 0 {
		t.Errorf("Expected non-empty hashes map, got empty")
	}
}

// TestApplyDeltaEmptyCommands verifies that ApplyDelta correctly handles an empty set of delta commands.
func TestApplyDeltaEmptyCommands(t *testing.T) {
	originalContent := []byte("Original content remains unchanged.")
	deltaCommands := []DeltaCommand{} // Empty delta commands

	originalFile, err := os.CreateTemp("", "original")
	if err != nil {
		t.Fatalf("Failed to create temporary original file: %v", err)
	}
	defer os.Remove(originalFile.Name()) // Clean up

	_, err = originalFile.Write(originalContent)
	if err != nil {
		t.Fatalf("Failed to write to temporary original file: %v", err)
	}
	originalFile.Close()

	outputFile, err := os.CreateTemp("", "output")
	if err != nil {
		t.Fatalf("Failed to create temporary output file: %v", err)
	}
	defer os.Remove(outputFile.Name()) // Clean up
	outputFile.Close()

	// Apply delta commands to the original file
	err = ApplyDelta(originalFile.Name(), deltaCommands, outputFile.Name())
	if err != nil {
		t.Errorf("ApplyDelta failed: %v", err)
	}

	// Read the content of the output file
	resultContent, err := os.ReadFile(outputFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temporary output file: %v", err)
	}

	// Since the deltaCommands is empty, we expect the output file to be empty
	expectedContent := []byte{} // Expecting an empty output file

	// Adjusting the assertion to reflect the expected outcome
	if !bytes.Equal(resultContent, expectedContent) {
		t.Errorf("Output file content does not match the expected empty content with empty delta commands.\nExpected an empty file\nGot: %s", resultContent)
	}
}

// TestApplyDeltaInvalidCommands verifies that ApplyDelta handles invalid delta commands gracefully.
func TestApplyDeltaInvalidCommands(t *testing.T) {
	originalContent := []byte("Some original content.")
	// A copy command with an invalid block index.
	deltaCommands := []DeltaCommand{
		{
			Command:    "copy",
			Position:   0,
			BlockIndex: -1, // Invalid block index
		},
	}

	originalFile, err := os.CreateTemp("", "original")
	if err != nil {
		t.Fatalf("Failed to create temporary original file: %v", err)
	}
	defer os.Remove(originalFile.Name()) // Clean up

	_, err = originalFile.Write(originalContent)
	if err != nil {
		t.Fatalf("Failed to write to temporary original file: %v", err)
	}
	originalFile.Close()

	outputFile, err := os.CreateTemp("", "output")
	if err != nil {
		t.Fatalf("Failed to create temporary output file: %v", err)
	}
	defer os.Remove(outputFile.Name()) // Clean up
	outputFile.Close()

	err = ApplyDelta(originalFile.Name(), deltaCommands, outputFile.Name())
	if err == nil {
		t.Errorf("Expected an error for invalid delta commands, but got none")
	}
}
