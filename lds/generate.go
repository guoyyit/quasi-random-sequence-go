package lds

// Generate produces n points of the configured sequence in one batch.
//
// The j-th returned point (0-based) is exactly the j-th point obtained by
// repeatedly calling Stream.Next with an equivalent configuration, because
// both paths call the same deterministic pointAt(j+1) function.
func Generate(cfg Config, n int) ([][]float64, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, ErrInvalidSamples
	}
	if n > maxGenerateSamples {
		return nil, ErrSamplesTooLarge
	}
	gen, err := newGenerator(cfg)
	if err != nil {
		return nil, err
	}
	points := make([][]float64, n)
	for j := 0; j < n; j++ {
		p := gen.pointAt(j + 1)
		point := make([]float64, cfg.Dim)
		copy(point, p)
		points[j] = point
	}
	return points, nil
}
