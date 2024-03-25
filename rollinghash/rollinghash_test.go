package rollinghash

import (
	"testing"
)

// TestNew ensures that a new RollingHash is initialized correctly.
func TestNew(t *testing.T) {
	windowSize := 5
	rh := New(windowSize)

	if rh.window != windowSize {
		t.Errorf("Expected window size of %d, got %d", windowSize, rh.window)
	}

	if rh.baseN != powMod(Base, windowSize-1, ModPrime) {
		t.Errorf("Expected baseN to be %d, got %d", powMod(Base, windowSize-1, ModPrime), rh.baseN)
	}
}

// TestHashData verifies that HashData correctly computes the initial hash for a given data window.
func TestHashData(t *testing.T) {
	data := []byte("hello")
	expectedHash := 0
	rh := New(len(data))

	// Manually calculate the expected hash for comparison.
	for _, b := range data {
		expectedHash = (expectedHash*Base + int(b)) % ModPrime
	}

	rh.HashData(data)

	if rh.GetHash() != expectedHash {
		t.Errorf("Expected hash of %d, got %d", expectedHash, rh.GetHash())
	}
}

// TestRoll verifies the rolling functionality of the RollingHash.
func TestRoll(t *testing.T) {
	data := []byte("hello")
	newByte := byte('a') // New byte to add
	rh := New(len(data))
	rh.HashData(data)

	// Simulate a roll operation: remove 'h', add 'a'.
	expectedHash := rh.GetHash()
	expectedHash = (expectedHash + ModPrime - int(data[0])*rh.baseN%ModPrime) % ModPrime
	expectedHash = (expectedHash*Base + int(newByte)) % ModPrime

	rh.Roll(data[0], newByte)

	if rh.GetHash() != expectedHash {
		t.Errorf("Expected hash of %d after rolling, got %d", expectedHash, rh.GetHash())
	}
}

// Helper function: powMod calculates (base^exp) % mod efficiently.
func powMod(base, exp, mod int) int {
	result := 1
	for exp > 0 {
		if exp%2 == 1 {
			result = (result * base) % mod
		}
		base = (base * base) % mod
		exp /= 2
	}
	return result
}
