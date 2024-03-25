package main

import (
	"fmt"
)

const (
	base     = 256       // Base value to use for hashing; 256 works well for binary files and text.
	modPrime = 805306457 // A large prime number for modulo operation to avoid hash collisions.
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

func main() {
	// Example usage
	windowSize := 4 // For demonstration, a small window size
	rh := NewRollingHash(windowSize)

	// Simulating adding bytes to the rolling hash
	data := []byte{'a', 'b', 'c', 'd'}
	for _, b := range data {
		rh.AddByte(b)
	}

	fmt.Printf("Initial hash: %d\n", rh.GetHash())

	// Now roll the window by removing 'a' and adding 'e'
	rh.Roll(data[0], 'e')

	fmt.Printf("Rolled hash: %d\n", rh.GetHash())
}
