package pinlib

import (
	"math/rand"
)

// Rng is a struct used to generate a random number generator from an initial seed
type Rng struct {
	rand *rand.Rand
}

// NewRng is used to create a new Rng struct
func NewRng(seed int64) *Rng {
	return &Rng{rand: rand.New(rand.NewSource(seed))}
}

// RandomNonceGenerator method is used to generate a 12 byte random nonce.
func (rng *Rng) RandomNonceGenerator() [12]byte {
	var p = [12]byte{0}
	rng.rand.Read(p[:])
	return p
}
