package lds

// Kind identifies the point-generation mechanism used by a sequence.
type Kind string

const (
	// Halton is a radical-inversion sequence with a distinct coprime base
	// (prime) per dimension.
	Halton Kind = "halton"

	// Kronecker is a Kronecker sequence n*alpha (mod 1) whose per-dimension
	// coordinates are the Kronecker–Richtmyer generalized golden ratios.
	Kronecker Kind = "kronecker"
)

// Options configures sequence generation.
type Options struct {
	// Kind selects the generation mechanism. Defaults to Halton.
	Kind Kind

	// Scramble enables randomization. When false Seed is ignored and the
	// generated points are the canonical (deterministic) sequence.
	Scramble bool

	// Seed is the master random seed used when Scramble is true. Equal seeds
	// (with all other parameters equal) reproduce the exact same point set.
	Seed uint64

	// Stream selects an independent sub-stream. Streams are derived from the
	// master Seed and produce mutually uncorrelated point sets. Stream must be
	// >= 0.
	Stream int64

	// StartIndex is the index of the first point produced by Generate; zero
	// corresponds to the first point of the sequence.
	StartIndex int64
}

// Sequence is a stateful, single-threaded low-discrepancy point stream.
// For concurrent use, create one Sequence per goroutine.
type Sequence interface {
	// Next returns the next point (length dim) and advances the stream.
	Next() ([]float64, error)

	// NextN returns the next n points as an n x dim matrix and advances the
	// stream. Repeated NextN/Next calls are identical to one batched call.
	NextN(n int) ([][]float64, error)

	// Reset moves the stream back to its initial index.
	Reset()

	// Dim reports the spatial dimension.
	Dim() int

	// Index reports how many points have already been produced.
	Index() int64
}

// MaxDim is the largest supported spatial dimension.
const MaxDim = 1024

// maxAbsoluteIndex is the largest allowed absolute generator index.
//
// Kronecker coordinates use the integer index as a 64-bit multiplicative
// operand, so the absolute index must be a non-negative int64; 2^63-1 is the
// inclusive maximum. Public position k maps to absolute index StartIndex+k+1.
const maxAbsoluteIndex int64 = (int64(1)<<62-1)*2 + 1 // 2^63 - 1
