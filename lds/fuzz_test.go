package lds

import "testing"

func FuzzGenerateParity(f *testing.F) {
	f.Add(3, 20, 0, uint64(0), int64(0))
	f.Add(1, 1, 1, uint64(42), int64(0))
	f.Add(7, 130, 1, uint64(1), int64(5))
	f.Fuzz(func(t *testing.T, dim, n int, kindIdx int, seed uint64, start int64) {
		// Restrict to legal parameter ranges; illegal ones are covered by
		// deterministic boundary tests and must only return errors.
		if dim <= 0 || dim > 20 || n <= 0 || n > 500 || start < 0 || start > 1000 {
			if _, err := Generate(dim, n, Options{}); err == nil &&
				(dim <= 0 || dim > MaxDim || n <= 0) {
				t.Fatalf("expected error for dim=%d n=%d", dim, n)
			}
			return
		}
		kind := Halton
		scramble := false
		switch kindIdx & 3 {
		case 0:
			kind = Halton
		case 1:
			kind = Kronecker
		case 2:
			kind, scramble = Halton, true
		case 3:
			kind, scramble = Kronecker, true
		}
		opts := Options{Kind: kind, Scramble: scramble, Seed: seed, StartIndex: start}

		batch, err := Generate(dim, n, opts)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		seq, err := NewSequence(dim, opts)
		if err != nil {
			t.Fatalf("NewSequence: %v", err)
		}
		inc, err := seq.NextN(n)
		if err != nil {
			t.Fatalf("NextN: %v", err)
		}
		if !pointsEqual(batch, inc) {
			t.Fatal("incremental != batch")
		}
		for _, p := range batch {
			for _, x := range p {
				if x <= 0 || x >= 1 {
					t.Fatalf("coordinate %v outside (0,1)", x)
				}
			}
		}
	})
}
