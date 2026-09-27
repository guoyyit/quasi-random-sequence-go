package lds

import "errors"

// Index convention: the public point at position k (k = 0,1,2,...) is the
// generator's integer point at (StartIndex + k + 1). Starting at 1 skips the
// degenerate all-zero origin point produced by both supported mechanisms at
// integer index 0 (the Halton radical inverse of 0 and the Kronecker value
// {0*alpha}). It also avoids the zero coordinate that a base-2 digit
// permutation would place there.

// Generate produces n points in the dim-dimensional unit cube [0,1)^dim
// according to opts. It is the stateless batch API.
func Generate(dim, n int, opts Options) (points [][]float64, err error) {
	opts, err = validate(dim, n, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Any unforeseen numeric/panic failure becomes an explicit error
		// instead of crashing the caller.
		if r := recover(); r != nil {
			err = errors.Join(ErrTooLarge, errors.New("lds: panic during generation"))
			points = nil
		}
	}()

	gen, err := newGenerator(dim, opts)
	if err != nil {
		return nil, err
	}
	points = make([][]float64, n)
	for k := 0; k < n; k++ {
		points[k] = gen.pointAt(opts.StartIndex + int64(k) + 1)
	}
	return points, nil
}

// NewSequence creates a stateful incremental point stream. Each call returns
// an independent object; for concurrent use, create one Sequence per goroutine.
func NewSequence(dim int, opts Options) (Sequence, error) {
	// A stream itself does not need a sample count, so validate dimensions
	// and options without one.
	if _, err := validate(dim, 1, opts); err != nil {
		return nil, err
	}
	gen, err := newGenerator(dim, opts)
	if err != nil {
		return nil, err
	}
	return &stream{gen: gen, dim: dim, start: opts.StartIndex}, nil
}

// newGenerator constructs the concrete mechanism from validated options.
func newGenerator(dim int, opts Options) (pointGenerator, error) {
	switch opts.Kind {
	case Halton, "":
		return newHalton(dim, opts)
	case Kronecker:
		return newKronecker(dim, opts)
	default:
		return nil, ErrUnsupported
	}
}

// pointGenerator is the common interface of concrete mechanisms.
type pointGenerator interface {
	pointAt(i int64) []float64
}

// stream wraps a pointGenerator with a mutable cursor.
type stream struct {
	gen   pointGenerator
	dim   int
	start int64
	index int64 // number of points already produced
}

// Next returns the next point and advances the stream.
func (s *stream) Next() ([]float64, error) {
	// Compare without computing start+index+1, which could overflow int64.
	if s.index < 0 || s.index > maxAbsoluteIndex-s.start-1 {
		return nil, errors.Join(ErrTooLarge, errors.New("lds: sequence index exceeded"))
	}
	absolute := s.start + s.index + 1
	p := s.gen.pointAt(absolute)
	s.index++
	return p, nil
}

// NextN returns the next n points and advances the stream.
func (s *stream) NextN(n int) ([][]float64, error) {
	if n <= 0 {
		return nil, errors.Join(ErrInvalidSamples, errors.New("lds: n must be positive"))
	}
	// Compare without computing the absolute last index, which could
	// overflow int64.
	if s.index < 0 || int64(n) < 0 || int64(n) > maxAbsoluteIndex-s.start-s.index {
		return nil, errors.Join(ErrTooLarge, errors.New("lds: sequence index exceeded"))
	}
	out := make([][]float64, n)
	for k := 0; k < n; k++ {
		out[k] = s.gen.pointAt(s.start + s.index + 1)
		s.index++
	}
	return out, nil
}

// Reset moves the stream back to its initial index.
func (s *stream) Reset() { s.index = 0 }

// Dim reports the spatial dimension.
func (s *stream) Dim() int { return s.dim }

// Index reports how many points have been produced since the last Reset.
func (s *stream) Index() int64 { return s.index }

// validate checks common parameters and normalizes options.
func validate(dim, n int, opts Options) (Options, error) {
	if dim <= 0 {
		return opts, errors.Join(ErrInvalidDimension, errors.New("lds: dim must be positive"))
	}
	if dim > MaxDim {
		return opts, errors.Join(ErrInvalidDimension, errors.New("lds: dim exceeds MaxDim"))
	}
	if n == 0 {
		return opts, errors.Join(ErrInvalidSamples, errors.New("lds: n must be positive (zero samples requested)"))
	}
	if n < 0 {
		return opts, errors.Join(ErrInvalidSamples, errors.New("lds: n must be positive (negative samples requested)"))
	}
	if opts.Stream < 0 {
		return opts, errors.Join(ErrInvalidSeed, errors.New("lds: stream must be >= 0"))
	}
	if opts.StartIndex < 0 {
		return opts, errors.Join(ErrInvalidSamples, errors.New("lds: StartIndex must be >= 0"))
	}
	if int64(n) > maxAbsoluteIndex || int64(n) > maxAbsoluteIndex-opts.StartIndex-1 {
		return opts, errors.Join(ErrTooLarge, errors.New("lds: sample range exceeds safe integer index"))
	}
	if opts.Kind == "" {
		opts.Kind = Halton
	}
	return opts, nil
}
