package lds

import "errors"

// Sentinel errors returned for invalid parameters. Callers can use errors.Is.
var (
	// ErrInvalidSamples is returned when the requested sample count is
	// non-positive, negative, or too large to be represented safely.
	ErrInvalidSamples = errors.New("lds: invalid sample count")

	// ErrInvalidDimension is returned when dim is zero, negative, or exceeds
	// the supported maximum dimension.
	ErrInvalidDimension = errors.New("lds: invalid dimension")

	// ErrInvalidSeed is returned when a scramble seed is invalid.
	ErrInvalidSeed = errors.New("lds: invalid scramble seed")

	// ErrUnsupported is returned for an unknown sequence kind.
	ErrUnsupported = errors.New("lds: unsupported sequence kind")

	// ErrTooLarge is returned when a computation would overflow or cannot be
	// completed safely with the given parameters.
	ErrTooLarge = errors.New("lds: parameter too large")
)
