package lds

import (
	"math/bits"
	"sync"
)

// permTables stores deterministic digit permutations used to scramble
// radical-inverse / digital sequences (Kocis-Owen style scrambling).
//
// Every coordinate (dimension d, digit position pos, base b) gets its own
// permutation of {0,...,b-1} derived solely from (seed, stream, d, pos, b).
// Different seeds derive disjoint permutation sets, hence different point
// sets, while identical parameters reproduce identical permutations bit for bit.
type permTables struct {
	seed   uint64
	stream int

	mu    sync.RWMutex
	cache map[uint64][]int
}

func newPermTables(seed uint64, stream int) *permTables {
	return &permTables{
		seed:   seed,
		stream: stream,
		cache:  make(map[uint64][]int),
	}
}

// perm returns the scrambling permutation for dimension d at digit position
// pos in the given base.
func (p *permTables) perm(base, d, pos int) []int {
	key := mixKey(p.seed, p.stream, base, d, pos)
	p.mu.RLock()
	perm, ok := p.cache[key]
	p.mu.RUnlock()
	if ok {
		return perm
	}

	perm = buildPermutation(base, key)
	p.mu.Lock()
	if existing, ok := p.cache[key]; ok {
		perm = existing
	} else {
		p.cache[key] = perm
	}
	p.mu.Unlock()
	return perm
}

// buildPermutation builds a Fisher-Yates shuffle of {0,...,base-1} driven by a
// deterministic splitmix64 stream seeded with key.
func buildPermutation(base int, key uint64) []int {
	perm := make([]int, base)
	for i := range perm {
		perm[i] = i
	}
	state := key
	for i := base - 1; i > 0; i-- {
		state = splitmix64(state)
		j := int(state % uint64(i+1))
		perm[i], perm[j] = perm[j], perm[i]
	}
	return perm
}

// mixKey folds all permutation-selecting parameters into a single uint64.
func mixKey(seed uint64, stream, base, d, pos int) uint64 {
	x := seed ^ uint64(0x9E3779B97F4A7C15)
	x = splitmix64(x ^ uint64(stream))
	x = splitmix64(x ^ uint64(base))
	x = splitmix64(x ^ uint64(d))
	x = splitmix64(x ^ uint64(pos))
	return x
}

// splitmix64 is the deterministic, well-distributed SplitMix64 bijection.
func splitmix64(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	z := x
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// interiorShift returns a strictly interior deterministic offset alpha in
// (0,1) added to a scrambled base-b coordinate that is an exact multiple of
// 1/den (den == b^K). The offset is:
//
//   - a multiple of 1/M where M is a power of two with 2 <= M <= 2^51/den,
//     so adding (A + alpha)/den rounds to a float64 strictly inside (0,1):
//     distinct integer fractions that map to the same float stay at least two
//     ulps apart after distinct shifts, and neither 0 nor 1 is reachable;
//   - derived solely from (seed, stream, base, d), so different streams (and
//     seeds) occupy disjoint fractional cosets and cannot share a point.
func interiorShift(seed uint64, stream, base, d, den int) float64 {
	// Largest power of two M = 1<<m with M*den <= 2^51.
	m := 51 - bits.Len(uint(den))
	if m < 1 {
		m = 1
	}
	x := seed ^ 0xD1B54A32D192ED03
	x = splitmix64(x ^ uint64(stream))
	x = splitmix64(x ^ uint64(base))
	x = splitmix64(x ^ uint64(d))
	slot := x & ((uint64(1) << uint(m)) - 1)
	return (float64(slot) + 0.5) / (float64(uint64(1) << uint(m)))
}
