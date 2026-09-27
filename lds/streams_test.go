package lds

import (
	"sync"
	"testing"
)

func TestStreamIndependence(t *testing.T) {
	for _, k := range kinds {
		const streams = 8
		const perStream = 128

		// Sequential reference per stream.
		refs := make([][][]float64, streams)
		for sid := 0; sid < streams; sid++ {
			p, err := Generate(2, perStream, Options{
				Kind: k, Scramble: true, Seed: 555, Stream: int64(sid),
			})
			if err != nil {
				t.Fatal(err)
			}
			refs[sid] = p
		}

		// No point set may repeat between streams.
		seen := make(map[[2]float64]int)
		for sid := 0; sid < streams; sid++ {
			for _, p := range refs[sid] {
				key := [2]float64{p[0], p[1]}
				if other, dup := seen[key]; dup {
					t.Fatalf("%s: duplicate point across streams %d and %d: %v", k, other, sid, key)
				}
				seen[key] = sid
			}
		}

		// Concurrent incremental generation with arbitrary interleaving must
		// match the per-stream sequential reference exactly.
		seqs := make([]Sequence, streams)
		for sid := range seqs {
			seq, err := NewSequence(2, Options{
				Kind: k, Scramble: true, Seed: 555, Stream: int64(sid),
			})
			if err != nil {
				t.Fatal(err)
			}
			seqs[sid] = seq
		}

		results := make([][][]float64, streams)
		var wg sync.WaitGroup
		for sid := 0; sid < streams; sid++ {
			wg.Add(1)
			go func(sid int) {
				defer wg.Done()
				got := make([][]float64, 0, perStream)
				for i := 0; i < perStream; i++ {
					// Mix Next and NextN to force arbitrary interleaving.
					if i%5 == 0 {
						chunk, err := seqs[sid].NextN(1)
						if err != nil {
							t.Error(err)
							return
						}
						got = append(got, chunk[0])
					} else {
						p, err := seqs[sid].Next()
						if err != nil {
							t.Error(err)
							return
						}
						got = append(got, p)
					}
				}
				results[sid] = got
			}(sid)
		}
		wg.Wait()

		for sid := 0; sid < streams; sid++ {
			if !pointsEqual(results[sid], refs[sid]) {
				t.Fatalf("%s stream %d: concurrent output differs from sequential", k, sid)
			}
		}
	}
}

func TestConcurrentBatchSafety(t *testing.T) {
	// The stateless batch API must be safe to call concurrently with the same
	// options and return identical matrices.
	for _, k := range kinds {
		const goroutines = 16
		ref, err := Generate(3, 200, Options{Kind: k, Scramble: true, Seed: 31})
		if err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		errs := make(chan error, goroutines)
		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p, err := Generate(3, 200, Options{Kind: k, Scramble: true, Seed: 31})
				if err != nil {
					errs <- err
					return
				}
				if !pointsEqual(p, ref) {
					errs <- errMismatch
				}
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Fatalf("%s: %v", k, err)
		}
	}
}

var errMismatch = stringError("concurrent batch results differ")

type stringError string

func (e stringError) Error() string { return string(e) }
