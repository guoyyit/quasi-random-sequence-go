package lds

import (
	"math"
	"sync"
	"testing"
)

// pointKey returns an exact bit-level key for a point, so duplicates are
// detected without any floating-point tolerance.
func pointKey(p []float64) string {
	b := make([]byte, 0, len(p)*8)
	var buf [8]byte
	for _, x := range p {
		v := math.Float64bits(x)
		for j := 0; j < 8; j++ {
			buf[j] = byte(v >> (uint(j) * 8))
		}
		b = append(b, buf[:]...)
	}
	return string(b)
}

// assertInteriorAndUnique fails the test if any coordinate touches 0 or 1 or
// if any point repeats within the set.
func assertInteriorAndUnique(t *testing.T, kind Kind, seed uint64, stream, dim, n int, pts [][]float64) {
	t.Helper()
	seen := make(map[string]bool, len(pts))
	for i, p := range pts {
		if len(p) != dim {
			t.Fatalf("kind=%d seed=%d stream=%d dim=%d: point %d has %d coords",
				kind, seed, stream, dim, i, len(p))
		}
		for _, x := range p {
			if math.IsNaN(x) || x <= 0.0 || x >= 1.0 {
				t.Fatalf("kind=%d seed=%d stream=%d dim=%d: boundary/NaN coordinate at %d: %v",
					kind, seed, stream, dim, i, p)
			}
		}
		k := pointKey(p)
		if seen[k] {
			t.Fatalf("kind=%d seed=%d stream=%d dim=%d: duplicate point at %d: %v",
				kind, seed, stream, dim, i, p)
		}
		seen[k] = true
	}
}

// Multi-seed, multi-dimension scan for both mechanisms: every scrambled point
// must lie strictly inside (0,1)^d and prefixes must be duplicate-free, via
// both batch generation and single-point incremental generation.
func TestScrambledInteriorAndUniqueScan(t *testing.T) {
	seeds := []uint64{0, 1, 2, 3, 7, 42, 99, 123456789, 20260927, ^uint64(0)}
	dims := []int{1, 2, 3, 4, 5, 8, 13}
	const n = 600
	for _, kind := range kinds {
		for _, seed := range seeds {
			for _, dim := range dims {
				cfg := Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed, Stream: 3}
				batch, err := Generate(cfg, n)
				if err != nil {
					t.Fatal(err)
				}
				assertInteriorAndUnique(t, kind, seed, 3, dim, n, batch)

				st, err := NewStream(cfg)
				if err != nil {
					t.Fatal(err)
				}
				incr := make([][]float64, n)
				for i := 0; i < n; i++ {
					incr[i], err = st.Next()
					if err != nil {
						t.Fatal(err)
					}
					if !pointEqual(incr[i], batch[i]) {
						t.Fatalf("kind=%d seed=%d dim=%d: incremental/batch divergence at %d: %v != %v",
							kind, seed, dim, i, incr[i], batch[i])
					}
				}
				assertInteriorAndUnique(t, kind, seed, 3, dim, n, incr)
			}
		}
	}
}

// Deep prefixes crossing several base^k digit-length boundaries must stay
// interior and pairwise distinct (regresses the short-index/long-index
// collapse of variable-length digit scrambling).
func TestScrambledDeepPrefixUnique(t *testing.T) {
	cases := []struct {
		kind Kind
		dim  int
	}{
		{Halton, 2},
		{Halton, 5},
		{Faure, 2},
		{Faure, 4},
	}
	const n = 50000
	for _, c := range cases {
		for _, seed := range []uint64{1, 777} {
			cfg := Config{Kind: c.kind, Dim: c.dim, Scramble: true, Seed: seed, Stream: 5}
			pts, err := Generate(cfg, n)
			if err != nil {
				t.Fatal(err)
			}
			assertInteriorAndUnique(t, c.kind, seed, 5, c.dim, n, pts)
		}
	}
}

// Under one seed, distinct stream numbers must generate disjoint point sets:
// no point produced by one stream may occur in any other stream.
func TestScrambledStreamsDisjointScan(t *testing.T) {
	seeds := []uint64{0, 1, 17, 424242}
	dims := []int{1, 2, 4, 7}
	streams := []int{0, 1, 2, 3, 9, 256}
	const n = 400
	for _, kind := range kinds {
		for _, seed := range seeds {
			for _, dim := range dims {
				pool := make(map[string]bool)
				for _, stream := range streams {
					cfg := Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed, Stream: stream}
					pts, err := Generate(cfg, n)
					if err != nil {
						t.Fatal(err)
					}
					assertInteriorAndUnique(t, kind, seed, stream, dim, n, pts)
					for i, p := range pts {
						k := pointKey(p)
						if pool[k] {
							t.Fatalf("kind=%d seed=%d dim=%d: point %d of stream %d already produced by another stream: %v",
								kind, seed, dim, i, stream, p)
						}
						pool[k] = true
					}
				}
			}
		}
	}
}

// Distinct seeds must produce distinct point sequences (and scrambled
// sequences must not equal the unscrambled one).
func TestScrambledSeedsDifferScan(t *testing.T) {
	seeds := []uint64{0, 1, 2, 50, 987654321, ^uint64(0)}
	dims := []int{1, 3, 6}
	const n = 300
	for _, kind := range kinds {
		for _, dim := range dims {
			var sets [][][]float64
			unscrambled, err := Generate(Config{Kind: kind, Dim: dim}, n)
			if err != nil {
				t.Fatal(err)
			}
			for _, seed := range seeds {
				cfg := Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed}
				pts, err := Generate(cfg, n)
				if err != nil {
					t.Fatal(err)
				}
				if pointSetsEqual(pts, unscrambled) {
					t.Fatalf("kind=%d dim=%d seed=%d: scrambled set equals unscrambled", kind, dim, seed)
				}
				for j, other := range sets {
					if pointSetsEqual(pts, other) {
						t.Fatalf("kind=%d dim=%d: seeds %d and %d produced identical sequences",
							kind, dim, seed, seeds[j])
					}
				}
				sets = append(sets, pts)
			}
		}
	}
}

// Interleaved concurrent Next calls must reproduce the sequential sequence
// for both mechanisms over several seeds and dimensions, including two
// independent streams advanced in lockstep.
func TestScrambledConcurrentInterleaveScan(t *testing.T) {
	dims := []int{1, 4, 9}
	seeds := []uint64{1, 31337}
	const total = 480
	for _, kind := range kinds {
		for _, dim := range dims {
			for _, seed := range seeds {
				cfg := Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed, Stream: 11}
				expected, err := Generate(cfg, total)
				if err != nil {
					t.Fatal(err)
				}
				st, err := NewStream(cfg)
				if err != nil {
					t.Fatal(err)
				}
				counts := make(map[string]int, total)
				var claimMu, countMu sync.Mutex
				var wg sync.WaitGroup
				for w := 0; w < 8; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for {
							claimMu.Lock()
							if st.Index() >= total {
								claimMu.Unlock()
								return
							}
							p, err := st.Next()
							claimMu.Unlock()
							if err != nil {
								return
							}
							countMu.Lock()
							counts[pointKey(p)]++
							countMu.Unlock()
						}
					}()
				}
				wg.Wait()
				if len(counts) != total {
					t.Fatalf("kind=%d seed=%d dim=%d: concurrent run produced %d distinct points, want %d",
						kind, seed, dim, len(counts), total)
				}
				for i, p := range expected {
					if counts[pointKey(p)] != 1 {
						t.Fatalf("kind=%d seed=%d dim=%d: expected point %d missing/duplicated under concurrency: %v",
							kind, seed, dim, i, p)
					}
				}

				other := Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed, Stream: 12}
				wantOther, err := Generate(other, 60)
				if err != nil {
					t.Fatal(err)
				}
				a, _ := NewStream(cfg)
				b, _ := NewStream(other)
				var gotA, gotB [][]float64
				for i := 0; i < 60; i++ {
					if i%2 == 0 {
						pa, _ := a.Next()
						pb, _ := b.Next()
						gotA = append(gotA, pa)
						gotB = append(gotB, pb)
					} else {
						pb, _ := b.Next()
						pa, _ := a.Next()
						gotA = append(gotA, pa)
						gotB = append(gotB, pb)
					}
				}
				wantA := expected[:60]
				if !pointSetsEqual(gotA, wantA) || !pointSetsEqual(gotB, wantOther) {
					t.Fatalf("kind=%d seed=%d dim=%d: lockstep streams drifted", kind, seed, dim)
				}
			}
		}
	}
}

// Scrambling must preserve low discrepancy: scrambled uniformity and
// correlation indicators stay within one order of magnitude of (or better
// than) the unscrambled baseline for both mechanisms.
func TestScrambledQualitySameOrder(t *testing.T) {
	dims := []int{2, 4, 8}
	const n = 1500
	for _, kind := range kinds {
		for _, dim := range dims {
			plain, err := Generate(Config{Kind: kind, Dim: dim}, n)
			if err != nil {
				t.Fatal(err)
			}
			mPlain, err := Assess(plain)
			if err != nil {
				t.Fatal(err)
			}
			for _, seed := range []uint64{1, 2, 3} {
				scr, err := Generate(Config{Kind: kind, Dim: dim, Scramble: true, Seed: seed}, n)
				if err != nil {
					t.Fatal(err)
				}
				mScr, err := Assess(scr)
				if err != nil {
					t.Fatal(err)
				}
				if mScr.L2StarDiscrepancy > 10*mPlain.L2StarDiscrepancy {
					t.Fatalf("kind=%d dim=%d seed=%d: scrambled discrepancy %v > 10x plain %v",
						kind, dim, seed, mScr.L2StarDiscrepancy, mPlain.L2StarDiscrepancy)
				}
				if mScr.MaxAbsCorrelation > 10*math.Max(mPlain.MaxAbsCorrelation, 1e-3) {
					t.Fatalf("kind=%d dim=%d seed=%d: scrambled max|corr| %v far above plain %v",
						kind, dim, seed, mScr.MaxAbsCorrelation, mPlain.MaxAbsCorrelation)
				}
			}
		}
	}
}
