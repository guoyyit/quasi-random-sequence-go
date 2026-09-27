package lds

import (
	"bytes"
	"math"
	"testing"
)

var kinds = []Kind{Halton, Kronecker}

func pointsEqual(a, b [][]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}

func TestBatchDeterminism(t *testing.T) {
	for _, k := range kinds {
		scrambleModes := []bool{false, true}
		for _, sc := range scrambleModes {
			opts := Options{Kind: k, Scramble: sc, Seed: 987654321}
			a, err := Generate(5, 64, opts)
			if err != nil {
				t.Fatal(err)
			}
			b, err := Generate(5, 64, opts)
			if err != nil {
				t.Fatal(err)
			}
			if !pointsEqual(a, b) {
				t.Fatalf("%s scramble=%v: repeated batch calls differ", k, sc)
			}
		}
	}
}

func TestIncrementalMatchesBatch(t *testing.T) {
	for _, k := range kinds {
		for _, sc := range []bool{false, true} {
			opts := Options{Kind: k, Scramble: sc, Seed: 7}
			want, err := Generate(4, 100, opts)
			if err != nil {
				t.Fatal(err)
			}

			// One NextN call.
			s1, _ := NewSequence(4, opts)
			oneShot, err := s1.NextN(100)
			if err != nil {
				t.Fatal(err)
			}
			if !pointsEqual(want, oneShot) {
				t.Fatalf("%s: single NextN differs from batch", k)
			}

			// One Next per point.
			s2, _ := NewSequence(4, opts)
			singles := make([][]float64, 100)
			for i := range singles {
				singles[i], err = s2.Next()
				if err != nil {
					t.Fatal(err)
				}
			}
			if !pointsEqual(want, singles) {
				t.Fatalf("%s: per-point Next differs from batch", k)
			}
		}
	}
}

func TestInterleavedChunksMatchSequential(t *testing.T) {
	// 37, 23, 40 interleaved must equal one 100-point batch.
	for _, k := range kinds {
		opts := Options{Kind: k, Scramble: true, Seed: 11}
		want, _ := Generate(3, 100, opts)

		s, _ := NewSequence(3, opts)
		var got [][]float64
		for _, c := range []int{37, 23, 40} {
			chunk, err := s.NextN(c)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, chunk...)
		}
		if !pointsEqual(got, want) {
			t.Fatalf("%s: chunked NextN differs from batch", k)
		}
		if s.Index() != 100 {
			t.Fatalf("index = %d, want 100", s.Index())
		}

		s.Reset()
		if s.Index() != 0 {
			t.Fatal("Reset did not zero index")
		}
		again, _ := s.NextN(100)
		if !pointsEqual(again, want) {
			t.Fatalf("%s: after Reset sequence differs", k)
		}
	}
}

func TestDifferentSeedsDiffer(t *testing.T) {
	for _, k := range kinds {
		pts := make(map[string]struct{})
		for seed := uint64(1); seed <= 8; seed++ {
			p, err := Generate(4, 50, Options{Kind: k, Scramble: true, Seed: seed})
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			for _, row := range p {
				for _, x := range row {
					bitsU := math.Float64bits(x)
					buf.WriteByte(byte(bitsU))
				}
			}
			key := buf.String()
			if _, dup := pts[key]; dup {
				t.Fatalf("%s: duplicate point set across seeds", k)
			}
			pts[key] = struct{}{}
		}
	}
}

func TestStartIndexIsSubsequence(t *testing.T) {
	for _, k := range kinds {
		full, _ := Generate(2, 30, Options{Kind: k})
		tail, _ := Generate(2, 10, Options{Kind: k, StartIndex: 20})
		if !pointsEqual(full[20:], tail) {
			t.Fatalf("%s: StartIndex slice mismatch", k)
		}
	}
}
