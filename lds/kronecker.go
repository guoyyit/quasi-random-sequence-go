package lds

import (
	"math"
	"math/big"
	"math/bits"
)

// kroneckerGenerator generates a (optionally shifted) Kronecker (rank-1
// lattice) sequence.
//
// Dimension d uses the Richtmyer (Wagner) generating-vector coordinate
//
//	alpha_d = { sqrt(p_d) },
//
// the fractional part of the square root of the d-th prime, evaluated to 256
// bits. Fractional parts of square roots of distinct integers are Q-linearly
// independent (along with 1), so the coordinates form a badly-approximable,
// low-discrepancy Kronecker sequence in any dimension, and a prime is
// available for every supported dimension without parameter tables.
//
// To avoid catastrophic cancellation in frac(i*alpha) for large i under
// float64 arithmetic, each alpha is converted once to a 64-bit fixed-point
// integer Q = round(alpha * 2^64), and the i-th value is
//
//	(i * Q) mod 2^64 / 2^64,
//
// evaluated with an exact 128-bit integer multiply (math/bits). This is a
// stateless O(1) computation per point, so incremental and batch callers run
// the identical code path and get bit-identical results.
type kroneckerGenerator struct {
	dim       int
	start     int64
	fixed     []uint64 // per-dimension alpha * 2^64 (rounded)
	shift     []uint64 // per-dimension Cranley–Patterson shift (fixed point)
	scrambled bool
}

// kroneckerPrec is the fixed precision (bits) used for sqrt(prime).
const kroneckerPrec = 256

// newKronecker creates a Kronecker generator. Parameters are assumed already
// validated by the caller.
func newKronecker(dim int, opts Options) (*kroneckerGenerator, error) {
	k := &kroneckerGenerator{
		dim:   dim,
		start: opts.StartIndex,
		fixed: make([]uint64, dim),
	}
	for d := 0; d < dim; d++ {
		k.fixed[d] = alphaToFixed(richtmyerAlpha(nthPrime(d)))
	}
	if opts.Scramble {
		k.scrambled = true
		k.shift = make([]uint64, dim)
		for d := 0; d < dim; d++ {
			seed := deriveSeed(opts.Seed, opts.Stream, purposeKronecker+uint64(d))
			k.shift[d] = fixedFromFloat(newRNG(seed).Float64())
		}
	}
	return k, nil
}

// pointAt returns the point at absolute sequence index i (i >= 0).
func (k *kroneckerGenerator) pointAt(i int64) []float64 {
	u := uint64(i)
	p := make([]float64, k.dim)
	for d := 0; d < k.dim; d++ {
		_, lo := bits.Mul64(u, k.fixed[d]) // {i*alpha_d} via multiplication mod 2^64
		if k.scrambled {
			lo += k.shift[d] // toroidal wrap is addition modulo 2^64
		}
		p[d] = float64(lo) * 0x1.0p-64
	}
	return p
}

// richtmyerAlpha returns { sqrt(p) }, the fractional part of the square root
// of the prime p, using fixed-precision, platform-independent arithmetic.
func richtmyerAlpha(p int) float64 {
	pf := new(big.Float).SetPrec(kroneckerPrec).SetInt64(int64(p))
	sqrtp := new(big.Float).SetPrec(kroneckerPrec).Sqrt(pf)
	f, _ := sqrtp.Float64()
	_, frac64 := math.Modf(f)
	return frac64
}

// alphaToFixed converts alpha in (0,1) to round(alpha*2^64).
func alphaToFixed(alpha float64) uint64 {
	bf := new(big.Float).SetPrec(kroneckerPrec).SetFloat64(alpha)
	scale := new(big.Float).SetPrec(kroneckerPrec).SetFloat64(0x1.0p64)
	scaled := new(big.Float).SetPrec(kroneckerPrec).Mul(bf, scale)
	bi, _ := scaled.Int(nil)
	return bi.Uint64()
}

// fixedFromFloat maps a float64 in [0,1) to a 64-bit fixed-point value.
func fixedFromFloat(x float64) uint64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return math.MaxUint64
	}
	bf := new(big.Float).SetPrec(kroneckerPrec).SetFloat64(x)
	scale := new(big.Float).SetPrec(kroneckerPrec).SetFloat64(0x1.0p64)
	scaled := new(big.Float).SetPrec(kroneckerPrec).Mul(bf, scale)
	bi, _ := scaled.Int(nil)
	return bi.Uint64()
}
