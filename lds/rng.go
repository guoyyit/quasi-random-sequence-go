package lds

// rng is a small deterministic 64-bit PRNG (SplitMix64) used for scrambling.
// Its behavior is fixed by specification, so output is bit-for-bit
// reproducible across Go versions and platforms, and it does not rely on the
// runtime's global random source.
type rng struct {
	state uint64
}

// newRNG creates a SplitMix64 stream seeded with seed.
func newRNG(seed uint64) *rng {
	return &rng{state: seed}
}

// Uint64 returns the next pseudo-random 64-bit value.
func (r *rng) Uint64() uint64 {
	r.state += 0x9e3779b97f4a7c15
	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// Uintn returns an unbiased pseudo-random integer in [0,n). n must be > 0.
func (r *rng) Uintn(n int) int {
	if n <= 1 {
		return 0
	}
	bound := uint64(n)
	// Smallest value such that values >= limit must be rejected, avoiding
	// modulo bias.
	limit := ^uint64(0) - ^uint64(0)%bound
	for {
		x := r.Uint64()
		if x < limit {
			return int(x % bound)
		}
	}
}

// Float64 returns a pseudo-random value in [0,1) using the top 53 bits.
func (r *rng) Float64() float64 {
	return float64(r.Uint64()>>11) * 0x1.0p-53
}

// deriveSeed derives an independent child seed from a master seed, a stream
// identifier and a purpose tag. Different (stream, purpose) pairs yield
// effectively independent child streams; equal inputs reproduce exactly.
func deriveSeed(master uint64, stream int64, purpose uint64) uint64 {
	// Convert the signed stream id to a uint64 with a bijection so that
	// negative ids (rejected at the public API) would still mix cleanly.
	s := uint64(stream)
	mix := master ^ s*0x9e3779b97f4a7c15 ^ purpose*0xd1b54a32d192ed03
	return newRNG(mix).Uint64()
}
