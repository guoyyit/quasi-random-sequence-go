package lds

import "errors"

// Sentinel errors returned by the package.
var (
	ErrInvalidKind        = errors.New("lds: unknown sequence kind")
	ErrInvalidDim         = errors.New("lds: dimension must be positive and within supported range")
	ErrInvalidSamples     = errors.New("lds: sample count must be positive")
	ErrSamplesTooLarge    = errors.New("lds: sample count too large")
	ErrInvalidStreamIndex = errors.New("lds: stream index must be non-negative")
	ErrInvalidPoints      = errors.New("lds: invalid point set")
)

const (
	// maxDim is the largest supported dimension. The 1000th prime (7919) is
	// used as the largest Halton base, which keeps generation practical.
	maxDim = 1000
	// maxGenerateSamples bounds memory used by a single Generate call.
	maxGenerateSamples = 10_000_000
	// maxAssessSamples bounds the O(n^2) discrepancy assessment.
	maxAssessSamples = 20_000
)

// validateConfig validates the common configuration fields.
func validateConfig(cfg Config) error {
	if cfg.Kind != Halton && cfg.Kind != Faure {
		return ErrInvalidKind
	}
	if cfg.Dim <= 0 || cfg.Dim > maxDim {
		return ErrInvalidDim
	}
	if cfg.Stream < 0 {
		return ErrInvalidStreamIndex
	}
	return nil
}
