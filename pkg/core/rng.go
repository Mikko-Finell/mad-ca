package core

import "math/rand/v2"

// RNG is a thin convenience wrapper around math/rand/v2 for deterministic seeding.
type RNG struct {
	r *rand.Rand
}

// NewRNG creates a deterministic RNG using the provided seed.
func NewRNG(seed int64) *RNG {
	return &RNG{r: rand.New(rand.NewPCG(uint64(seed), 0))}
}

// Bool returns a random boolean value.
func (r *RNG) Bool() bool {
	return r.r.IntN(2) == 1
}

// Uint8n returns a random uint8 in [0, n).
func (r *RNG) Uint8n(n uint8) uint8 {
	if n == 0 {
		return 0
	}
	return uint8(r.r.IntN(int(n)))
}

// FillBinary fills the buffer with 0/1 values using the RNG.
func FillBinary(r *rand.Rand, buf []uint8) {
	for i := range buf {
		buf[i] = uint8(r.IntN(2))
	}
}

// Source exposes the underlying rand.Rand for advanced use.
func (r *RNG) Source() *rand.Rand { return r.r }
