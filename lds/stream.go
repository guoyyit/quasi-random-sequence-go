package lds

import "sync"

// generator produces the deterministic i-th point of a sequence.
type generator interface {
	pointAt(i int) []float64
}

// newGenerator constructs the generator for cfg.
func newGenerator(cfg Config) (generator, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	switch cfg.Kind {
	case Halton:
		return newHalton(cfg)
	case Faure:
		return newFaure(cfg)
	default:
		return nil, ErrInvalidKind
	}
}

// stream is a thread-safe incremental cursor over a deterministic generator.
type stream struct {
	mu    sync.Mutex
	cfg   Config
	dim   int
	gen   generator
	index int // number of points already emitted; next point is index+1
}

// Next returns the next point of the stream. The returned slice is owned by
// the caller and may be modified.
func (s *stream) Next() ([]float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.index + 1
	if next > maxGenerateSamples {
		return nil, ErrSamplesTooLarge
	}
	p := s.gen.pointAt(next)
	s.index = next
	out := make([]float64, s.dim)
	copy(out, p)
	return out, nil
}

func (s *stream) Index() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.index
}

// NewStream validates the configuration and creates an incremental generator.
// Each call with the same parameters returns an independent stream whose state
// does not interfere with any other stream; distinct Stream values select
// distinct scrambled sub-streams.
func NewStream(cfg Config) (Stream, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	gen, err := newGenerator(cfg)
	if err != nil {
		return nil, err
	}
	return &stream{cfg: cfg, dim: cfg.Dim, gen: gen}, nil
}
