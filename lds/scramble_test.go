package lds

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

// scrambleSeeds and scrambleDims define the scan coverage. The scan runs both
// sequence mechanisms over many seeds and dimensions to validate the strict
// internal guarantee under a wide parameter range.
var scrambleSeeds = []uint64{1, 2, 42, 12345, 20260927}
var scrambleDims = []int{1, 2, 3, 4, 5, 8, 16}

// pointKey renders a point as a string key for exact (bit-for-bit) comparison.
func pointKey(p []float64) string { return fmt.Sprintf("%v", p) }

// TestScrambledStrictlyInside: every scrambled point, under every seed,
// dimension and sequence kind, must lie strictly inside the unit hypercube.
// No coordinate may equal 0 or 1.
func TestScrambledStrictlyInside(t *testing.T) {
	for _, kind := range kinds {
		for _, dim := range scrambleDims {
			for _, seed := range scrambleSeeds {
				cfg := Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed}
				pts, err := Generate(cfg, 200)
				if err != nil {
					t.Fatalf("%v dim=%d seed=%d: %v", kind, dim, seed, err)
				}
				for i, p := range pts {
					for _, x := range p {
						if x <= 0.0 || x >= 1.0 || math.IsNaN(x) || math.IsInf(x, 0) {
							t.Fatalf("%v dim=%d seed=%d point %d: coordinate out of (0,1): %v",
								kind, dim, seed, i, p)
						}
					}
				}
			}
		}
	}
}

// TestScrambledNoDuplicates: no two points within a single scrambled point set
// may be identical, for any seed, dimension or sequence kind.
func TestScrambledNoDuplicates(t *testing.T) {
	for _, kind := range kinds {
		for _, dim := range scrambleDims {
			for _, seed := range scrambleSeeds {
				cfg := Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed}
				pts, err := Generate(cfg, 200)
				if err != nil {
					t.Fatalf("%v dim=%d seed=%d: %v", kind, dim, seed, err)
				}
				seen := make(map[string]bool, len(pts))
				for i, p := range pts {
					key := pointKey(p)
					if seen[key] {
						t.Fatalf("%v dim=%d seed=%d: duplicate point at index %d: %v",
							kind, dim, seed, i, p)
					}
					seen[key] = true
				}
			}
		}
	}
}

// TestScrambledCrossStream: distinct streams under the same seed must not
// produce identical points. The shared-point check is restricted to dim >= 4
// where the value space is large enough that a chance collision is negligible;
// for smaller dimensions we still assert the sets are not identical.
func TestScrambledCrossStream(t *testing.T) {
	for _, kind := range kinds {
		for _, dim := range scrambleDims {
			for _, seed := range scrambleSeeds {
				cfg0 := Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed, Stream: 0}
				cfg1 := Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed, Stream: 1}
				p0, err := Generate(cfg0, 200)
				if err != nil {
					t.Fatalf("%v dim=%d seed=%d stream0: %v", kind, dim, seed, err)
				}
				p1, err := Generate(cfg1, 200)
				if err != nil {
					t.Fatalf("%v dim=%d seed=%d stream1: %v", kind, dim, seed, err)
				}
				if pointSetsEqual(p0, p1) {
					t.Fatalf("%v dim=%d seed=%d: streams 0 and 1 yield identical sets",
						kind, dim, seed)
				}
				if dim >= 4 {
					set := make(map[string]bool, len(p0))
					for _, p := range p0 {
						set[pointKey(p)] = true
					}
					for _, p := range p1 {
						if set[pointKey(p)] {
							t.Fatalf("%v dim=%d seed=%d: shared point across streams: %v",
								kind, dim, seed, p)
						}
					}
				}
			}
		}
	}
}

// TestScrambledCrossSeed: distinct seeds must not produce identical point
// sets. The shared-point check is restricted to dim >= 4 for the same
// collision-negligibility reason as TestScrambledCrossStream.
func TestScrambledCrossSeed(t *testing.T) {
	for _, kind := range kinds {
		for _, dim := range scrambleDims {
			for si := 0; si < len(scrambleSeeds); si++ {
				for sj := si + 1; sj < len(scrambleSeeds); sj++ {
					cfgI := Config{Kind: kind, Dim: dim, Scramble: true, Seed: scrambleSeeds[si]}
					cfgJ := Config{Kind: kind, Dim: dim, Scramble: true, Seed: scrambleSeeds[sj]}
					pi, err := Generate(cfgI, 200)
					if err != nil {
						t.Fatalf("%v dim=%d seed=%d: %v", kind, dim, scrambleSeeds[si], err)
					}
					pj, err := Generate(cfgJ, 200)
					if err != nil {
						t.Fatalf("%v dim=%d seed=%d: %v", kind, dim, scrambleSeeds[sj], err)
					}
					if pointSetsEqual(pi, pj) {
						t.Fatalf("%v dim=%d: seeds %d and %d yield identical sets",
							kind, dim, scrambleSeeds[si], scrambleSeeds[sj])
					}
					if dim >= 4 {
						set := make(map[string]bool, len(pi))
						for _, p := range pi {
							set[pointKey(p)] = true
						}
						for _, p := range pj {
							if set[pointKey(p)] {
								t.Fatalf("%v dim=%d: shared point across seeds %d and %d: %v",
									kind, dim, scrambleSeeds[si], scrambleSeeds[sj], p)
							}
						}
					}
				}
			}
		}
	}
}

// TestScrambledIncrementalEqualsBatch: incremental Next must match batch
// Generate point for point under scrambling, for both sequence mechanisms.
func TestScrambledIncrementalEqualsBatch(t *testing.T) {
	for _, kind := range kinds {
		for _, dim := range []int{1, 4, 8} {
			for _, seed := range []uint64{1, 42} {
				cfg := Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed}
				batch, err := Generate(cfg, 300)
				if err != nil {
					t.Fatalf("%v dim=%d seed=%d: %v", kind, dim, seed, err)
				}
				st, err := NewStream(cfg)
				if err != nil {
					t.Fatalf("%v dim=%d seed=%d: %v", kind, dim, seed, err)
				}
				for j := 0; j < len(batch); j++ {
					got, err := st.Next()
					if err != nil {
						t.Fatalf("%v dim=%d seed=%d point %d: %v", kind, dim, seed, j, err)
					}
					if !pointEqual(got, batch[j]) {
						t.Fatalf("%v dim=%d seed=%d point %d mismatch:\n got %v\nwant %v",
							kind, dim, seed, j, got, batch[j])
					}
				}
			}
		}
	}
}

// TestScrambledConcurrentInterleaving: interleaved concurrent Next calls on a
// shared stream must reproduce the sequential scrambled sequence, and
// independent streams advanced concurrently must stay isolated.
func TestScrambledConcurrentInterleaving(t *testing.T) {
	for _, kind := range kinds {
		cfg := Config{Kind: kind, Dim: 5, Scramble: true, Seed: 321}
		expected, err := Generate(cfg, 400)
		if err != nil {
			t.Fatalf("%v: %v", kind, err)
		}

		numWorkers := 8
		st, err := NewStream(cfg)
		if err != nil {
			t.Fatalf("%v: %v", kind, err)
		}
		var mu sync.Mutex
		collected := make([][]float64, 400)
		var wg sync.WaitGroup
		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					mu.Lock()
					idx := st.Index()
					if idx >= 400 {
						mu.Unlock()
						return
					}
					p, err := st.Next()
					mu.Unlock()
					if err != nil {
						t.Error(err)
						return
					}
					collected[idx] = p
				}
			}()
		}
		wg.Wait()
		if !pointSetsEqual(collected, expected) {
			t.Fatalf("%v: interleaved concurrent sequence differs from sequential", kind)
		}

		// Two streams advanced concurrently with interwoven calls stay isolated.
		a, _ := NewStream(cfg)
		b, _ := NewStream(withStream(cfg, 7))
		wantA, _ := Generate(cfg, 50)
		wantB, _ := Generate(withStream(cfg, 7), 50)
		var ga, gb [][]float64
		for i := 0; i < 50; i++ {
			if i%2 == 0 {
				p, _ := a.Next()
				ga = append(ga, p)
				p, _ = b.Next()
				gb = append(gb, p)
			} else {
				p, _ := b.Next()
				gb = append(gb, p)
				p, _ = a.Next()
				ga = append(ga, p)
			}
		}
		if !pointSetsEqual(ga, wantA) || !pointSetsEqual(gb, wantB) {
			t.Fatalf("%v: interleaved independent streams drifted", kind)
		}
	}
}

// TestScrambledLowDiscrepancy: scrambling must not degrade the low-discrepancy
// property by an order of magnitude. The scrambled L2* discrepancy must stay
// in the same ballpark as the unscrambled one.
func TestScrambledLowDiscrepancy(t *testing.T) {
	for _, kind := range kinds {
		for _, dim := range []int{2, 4, 8} {
			uPts, err := Generate(Config{Kind: kind, Dim: dim}, 500)
			if err != nil {
				t.Fatalf("%v dim=%d unscrambled: %v", kind, dim, err)
			}
			uMet, err := Assess(uPts)
			if err != nil {
				t.Fatalf("%v dim=%d unscrambled: %v", kind, dim, err)
			}
			for _, seed := range []uint64{1, 42} {
				sPts, err := Generate(Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed}, 500)
				if err != nil {
					t.Fatalf("%v dim=%d seed=%d: %v", kind, dim, seed, err)
				}
				sMet, err := Assess(sPts)
				if err != nil {
					t.Fatalf("%v dim=%d seed=%d: %v", kind, dim, seed, err)
				}
				// Scrambled discrepancy must not exceed 5x the unscrambled one
				// (same order of magnitude, no order-of-magnitude degradation).
				if sMet.L2StarDiscrepancy > 5*uMet.L2StarDiscrepancy {
					t.Fatalf("%v dim=%d seed=%d: scrambled discrepancy %.6f far exceeds unscrambled %.6f",
						kind, dim, seed, sMet.L2StarDiscrepancy, uMet.L2StarDiscrepancy)
				}
				// Correlations must remain small.
				if sMet.MaxAbsCorrelation > 0.1 {
					t.Fatalf("%v dim=%d seed=%d: max|corr| too high: %v",
						kind, dim, seed, sMet.MaxAbsCorrelation)
				}
			}
		}
	}
}
