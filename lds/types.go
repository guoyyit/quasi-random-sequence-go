// Package lds provides reproducible, locally generated low-discrepancy
// (quasi-random) sequences in the unit hypercube and deterministic quality
// assessment of their uniformity.
package lds

// Kind selects the generation mechanism of a sequence.
type Kind int

const (
	// Halton is the radical-inverse sequence based on per-dimension prime bases.
	Halton Kind = iota
	// Faure is the digital net sequence built from binomial matrices over a prime base.
	Faure
)

// Config configures a sequence generator.
type Config struct {
	// Kind selects the generation mechanism.
	Kind Kind
	// Dim is the number of dimensions of the unit hypercube.
	Dim int
	// Scramble enables reproducible digit scrambling when true.
	Scramble bool
	// Seed is the scrambling seed (used only when Scramble is true).
	Seed uint64
	// Stream selects an independent deterministic sub-stream.
	Stream int
}

// Stream is a stateful, single-point incremental generator.
//
// A Stream is safe for concurrent use by multiple goroutines: Next returns
// points in the logical sequence order regardless of how calls are
// interleaved with concurrent Generate calls using equivalent parameters.
type Stream interface {
	// Next returns the next point of the stream.
	Next() ([]float64, error)
	// Index returns the number of points already produced.
	Index() int
}

// Metrics holds deterministic quality indicators for a point set.
type Metrics struct {
	// N is the number of points assessed.
	N int
	// Dim is the dimension of the points.
	Dim int
	// L2StarDiscrepancy is the Warnock L2 star discrepancy (smaller is better).
	L2StarDiscrepancy float64
	// MaxAbsCorrelation is the largest absolute off-diagonal Pearson
	// correlation coefficient between coordinate pairs (smaller is better).
	MaxAbsCorrelation float64
	// MeanAbsCorrelation is the mean absolute off-diagonal Pearson correlation.
	MeanAbsCorrelation float64
}
