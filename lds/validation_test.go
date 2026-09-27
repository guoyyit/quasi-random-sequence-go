package lds

import (
	"errors"
	"testing"
)

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		dim  int
		n    int
		opts Options
		want error
	}{
		{"zero samples", 2, 0, Options{}, ErrInvalidSamples},
		{"negative samples", 2, -5, Options{}, ErrInvalidSamples},
		{"zero dimension", 0, 10, Options{}, ErrInvalidDimension},
		{"negative dimension", -3, 10, Options{}, ErrInvalidDimension},
		{"dimension over max", MaxDim + 1, 10, Options{}, ErrInvalidDimension},
		{"negative stream", 2, 10, Options{Stream: -1}, ErrInvalidSeed},
		{"negative start index", 2, 10, Options{StartIndex: -1}, ErrInvalidSamples},
		{"sample range overflow", 2, int(maxAbsoluteIndex), Options{}, ErrTooLarge},
		{"unsupported kind", 2, 10, Options{Kind: "sobol"}, ErrUnsupported},
	}
	// Sample-count cases only apply to the batch API; NewSequence has no
	// sample-count argument.
	batchOnly := map[string]bool{
		"zero samples": true, "negative samples": true, "sample range overflow": true,
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Generate(tc.dim, tc.n, tc.opts)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Generate: want error %v, got %v", tc.want, err)
			}
			if !batchOnly[tc.name] {
				_, err = NewSequence(tc.dim, tc.opts)
				if tc.name != "unsupported kind" && !errors.Is(err, tc.want) {
					t.Fatalf("NewSequence: want error %v, got %v", tc.want, err)
				}
			}
		})
	}
}

func TestSequenceIncrementalErrors(t *testing.T) {
	s, err := NewSequence(2, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.NextN(0); !errors.Is(err, ErrInvalidSamples) {
		t.Fatalf("NextN(0): got %v", err)
	}
	if _, err := s.NextN(-3); !errors.Is(err, ErrInvalidSamples) {
		t.Fatalf("NextN(-3): got %v", err)
	}
}

func TestHugeDimensionBoundary(t *testing.T) {
	// MaxDim itself must be supported.
	pts, err := Generate(MaxDim, 1, Options{})
	if err != nil {
		t.Fatalf("MaxDim should be supported: %v", err)
	}
	if len(pts[0]) != MaxDim {
		t.Fatalf("got %d coordinates", len(pts[0]))
	}
	for _, x := range pts[0] {
		if x <= 0 || x >= 1 {
			t.Fatalf("coordinate outside (0,1): %v", x)
		}
	}
}

func TestKroneckerHugeDimension(t *testing.T) {
	pts, err := Generate(MaxDim, 2, Options{Kind: Kronecker})
	if err != nil {
		t.Fatalf("Kronecker MaxDim: %v", err)
	}
	if len(pts) != 2 || len(pts[0]) != MaxDim {
		t.Fatalf("bad shape")
	}
}

func TestLargeButSafeSampleCount(t *testing.T) {
	// A few ten thousand points must not overflow and should be fast.
	const n = 20000
	pts, err := Generate(2, n, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != n {
		t.Fatalf("got %d points", len(pts))
	}
	// No duplicates in the first coordinate (base-2 van der Corput).
	seen := make(map[float64]struct{}, n)
	for _, p := range pts {
		seen[p[0]] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("got %d distinct first coordinates, want %d", len(seen), n)
	}
}
