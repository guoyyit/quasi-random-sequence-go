package lds

import (
	"math"
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

	tailMu sync.Mutex
	tails  map[uint64]float64
}

func newPermTables(seed uint64, stream int) *permTables {
	return &permTables{
		seed:   seed,
		stream: stream,
		cache:  make(map[uint64][]int),
		tails:  make(map[uint64]float64),
	}
}

// tail returns the infinite Owen-scrambling tail for digit positions at and
// above start:
//
//	T(start) = sum_{j=start}^{inf} perm_j(0) / base^{j+1}.
//
// The tail is what keeps a scrambled point strictly inside the unit
// hypercube: even when every leading digit permutes to 0, the tail is a
// strictly positive random value, so the coordinate never collapses to 0 and
// distinct indices never collide. The result is deterministic and cached per
// (seed, stream, base, dim, start).
func (p *permTables) tail(base, dim, start int) float64 {
	key := mixKey(p.seed, p.stream, base, dim, start)
	p.tailMu.Lock()
	defer p.tailMu.Unlock()
	if t, ok := p.tails[key]; ok {
		return t
	}
	t := 0.0
	weight := math.Pow(1.0/float64(base), float64(start+1))
	tailBound := weight
	for j := start; j < start+200; j++ {
		perm := p.perm(base, dim, j)
		t += float64(perm[0]) * weight
		if t+tailBound == t { // remaining tail is below 0.5 ulp of t
			break
		}
		weight /= float64(base)
		tailBound = weight
	}
	p.tails[key] = t
	return t
}

// scrambledValue computes the full Owen-scrambled value for the output digit
// vector y (least significant digit first). The first len(y) digits are
// permuted; the infinite tail (positions >= len(y), whose unpermuted digits
// are all 0) is added as well. The tail guarantees the result is never
// exactly 0 and distinct indices never collide, even for base 2 where the
// leading digit permutation alone would be trivial.
func scrambledValue(perms *permTables, base, dim int, y []int) float64 {
	k := len(y)
	var num, den int64 = 0, 1
	for j := 0; j < k; j++ {
		num = num*int64(base) + int64(perms.perm(base, dim, j)[y[j]])
		den *= int64(base)
	}
	sum := float64(num) / float64(den)
	sum += perms.tail(base, dim, k)
	return sum
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
