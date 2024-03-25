package rollinghash

const (
	Base     = 256       // Base is used for the rolling hash calculation of the number of characters in the input charset.
	ModPrime = 805306457 // ModPrime is a large prime number used to avoid overflow and ensure hash uniqueness.
)

type RollingHash struct {
	hash   int
	window int
	first  int
	baseN  int
}

// New initializes a new RollingHash with a given window size.
func New(window int) *RollingHash {
	rh := &RollingHash{window: window}
	rh.baseN = 1 // Initial baseN will be raised to the power of (window-1).
	for i := 0; i < window-1; i++ {
		rh.baseN = (rh.baseN * Base) % ModPrime // Precompute baseN to optimize the roll operation.
	}
	return rh
}

// HashData calculates the hash of the initial window of data.
func (rh *RollingHash) HashData(data []byte) {
	rh.hash = 0
	for i, b := range data {
		if i == 0 {
			rh.first = int(b)
		}
		rh.AddByte(b)
	}
}

// AddByte incorporates a new byte into the hash value, used during initialization and rolling.
func (rh *RollingHash) AddByte(b byte) {
	rh.hash = (rh.hash*Base + int(b)) % ModPrime
}

// RemoveByte updates the hash by removing the contribution of the oldest byte in the window.
func (rh *RollingHash) RemoveByte(b byte) {
	rh.hash = (rh.hash + ModPrime - rh.first*rh.baseN%ModPrime) % ModPrime
	rh.first = int(b)
}

// Roll updates the hash to reflect removing the oldest byte and adding a new one.
func (rh *RollingHash) Roll(oldByte, newByte byte) {
	rh.RemoveByte(oldByte)
	rh.AddByte(newByte)
}

// GetHash returns the current hash value.
func (rh *RollingHash) GetHash() int {
	return rh.hash
}
