// rollinghash/rollinghash.go

package rollinghash

const (
	Base     = 256
	ModPrime = 805306457
)

type RollingHash struct {
	hash   int
	window int
	first  int
	baseN  int
}

func New(window int) *RollingHash {
	rh := &RollingHash{window: window}
	rh.baseN = 1
	for i := 0; i < window-1; i++ {
		rh.baseN = (rh.baseN * Base) % ModPrime
	}
	return rh
}

func (rh *RollingHash) HashData(data []byte) {
	rh.hash = 0
	for i, b := range data {
		if i == 0 {
			rh.first = int(b)
		}
		rh.AddByte(b)
	}
}

func (rh *RollingHash) AddByte(b byte) {
	rh.hash = (rh.hash*Base + int(b)) % ModPrime
}

func (rh *RollingHash) RemoveByte(b byte) {
	rh.hash = (rh.hash + ModPrime - rh.first*rh.baseN%ModPrime) % ModPrime
	rh.first = int(b)
}

func (rh *RollingHash) Roll(oldByte, newByte byte) {
	rh.RemoveByte(oldByte)
	rh.AddByte(newByte)
}

func (rh *RollingHash) GetHash() int {
	return rh.hash
}
