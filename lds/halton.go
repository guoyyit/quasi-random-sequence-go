package lds

// Purpose tags mix independent RNG sub-streams; keeping them in one place
// makes the scrambling key derivation auditable.
const (
	purposeHalton    uint64 = 0x48616c746f6e // "Halton"
	purposeKronecker uint64 = 0x4b726f6e6563 // "Kronec"
)

// haltonGenerator generates a (optionally scrambled) Halton sequence.
//
// Dimension d uses the d-th prime as its radical-inversion base, so all
// per-dimension bases are pairwise coprime. The scrambled variant is a
// Faure–Tezuka style digital permutation scramble: every digit position is
// mapped through a fixed per-dimension permutation of the base alphabet,
// including the (infinitely many) leading-zero positions. Such a permutation
// maps each b-adic elementary interval bijectively onto another interval of
// equal volume, preserving the low-discrepancy order, while making the points
// seed-keyed and avoiding the unscrambled origin point.
type haltonGenerator struct {
	dim   int
	start int64
	bases []int
	// perms[d][k] is the digit permutation at digit position k in dimension d
	// (Owen/Faure–Tezuka per-position scrambling); nil when unscrambled.
	perms [][][]int
}

// newHalton creates a Halton generator. Parameters are assumed already
// validated by the caller.
func newHalton(dim int, opts Options) (*haltonGenerator, error) {
	h := &haltonGenerator{
		dim:   dim,
		start: opts.StartIndex,
		bases: make([]int, dim),
	}
	for d := 0; d < dim; d++ {
		h.bases[d] = nthPrime(d)
	}
	if opts.Scramble {
		h.perms = make([][][]int, dim)
		for d := 0; d < dim; d++ {
			// Enough positions for any 64-bit index plus the leading-zero
			// tail; each position gets its own independent key.
			perms := make([][]int, haltonMaxPositions)
			for k := range perms {
				seed := deriveSeed(opts.Seed, opts.Stream,
					purposeHalton^(uint64(d)*0x100000001b3)^(uint64(k)*0x9e3779b97f4a7c15))
				perm := digitPermutation(h.bases[d], newRNG(seed))
				// Ensure perm[0] is neither 0 (origin degeneracy) nor b-1
				// (which would put a leading-zero tail at the boundary 1).
				if perm[0] == 0 || perm[0] == h.bases[d]-1 {
					target := 1
					if h.bases[d] > 2 && perm[target] == 0 {
						target = 2
					}
					perm[0], perm[target] = perm[target], perm[0]
				}
				perms[k] = perm
			}
			h.perms[d] = perms
		}
	}
	return h, nil
}

// pointAt returns the point at absolute sequence index i (i >= 0).
func (h *haltonGenerator) pointAt(i int64) []float64 {
	p := make([]float64, h.dim)
	for d := 0; d < h.dim; d++ {
		var perms [][]int
		if h.perms != nil {
			perms = h.perms[d]
		}
		p[d] = radicalInverse(i, h.bases[d], perms)
	}
	return p
}

// haltonMaxPositions bounds how many base-b digit positions are precomputed.
// Any sequence index below 2^63 uses fewer than 64 base-2 positions.
const haltonMaxPositions = 64

// radicalInverse returns the (possibly scrambled) radical inverse of the
// non-negative integer i in the given base.
//
// Unscrambled (perms == nil): the classical van der Corput value in [0,1).
//
// Scrambled (Faure–Tezuka/Owen digital scramble): the digit at position k is
// mapped through perms[k]; every leading-zero position above the highest digit
// contributes perms[k][0]. Because the position-0 permutation sends 0 to a
// value strictly inside 1..b-2 (see newHalton), the result lies strictly in
// (0,1).
func radicalInverse(i int64, base int, perms [][]int) float64 {
	f := 1.0 / float64(base)
	result := 0.0
	k := 0
	for i > 0 {
		digit := int(i % int64(base))
		if perms != nil {
			digit = perms[k][digit]
		}
		result += float64(digit) * f
		f /= float64(base)
		i /= int64(base)
		k++
	}
	if perms != nil {
		// Leading-zero positions from k up to the precomputed limit.
		// sum_{j>=k+1} b^{-j} = f_k/(b-1); truncate at haltonMaxPositions,
		// whose omitted tail is below 2^{-64}.
		for j := k; j < haltonMaxPositions; j++ {
			result += float64(perms[j][0]) * f
			f /= float64(base)
		}
	}
	return result
}

// digitPermutation returns an unbiased permutation of 0..b-1, generated with
// a Fisher–Yates shuffle driven by r.
func digitPermutation(b int, r *rng) []int {
	p := make([]int, b)
	for i := range p {
		p[i] = i
	}
	for i := len(p) - 1; i > 0; i-- {
		j := r.Uintn(i + 1)
		p[i], p[j] = p[j], p[i]
	}
	return p
}
