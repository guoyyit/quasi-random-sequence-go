// Command qualitydemo generates low-discrepancy point sets with two
// mechanism-different sequences (Halton and Kronecker), optionally scrambles
// them, and prints deterministic uniformity/correlation quality metrics.
//
// Usage:
//
//	go run ./examples/qualitydemo [-n 1000] [-d 3] [-bins 16] [-seed 1234] [-scramble]
//
// Output is fully reproducible for identical flags; no network or external
// services are used.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"

	"github.com/guoyyit/quasi-random-sequence-go/lds"
)

func main() {
	n := flag.Int("n", 1000, "number of samples")
	dim := flag.Int("d", 3, "spatial dimension")
	bins := flag.Int("bins", 16, "uniform grid bins for chi-square")
	seed := flag.Uint64("seed", 1234, "scramble seed")
	scramble := flag.Bool("scramble", false, "enable deterministic scrambling")
	flag.Parse()

	opts := lds.Options{Scramble: *scramble, Seed: *seed}

	fmt.Printf("quality report  n=%d dim=%d bins=%d scramble=%v seed=%d\n\n",
		*n, *dim, *bins, *scramble, *seed)
	fmt.Printf("%-10s %12s %12s %12s %10s %10s\n",
		"sequence", "L2StarDisc", "max|corr|", "chiMax", "edges", "outRange")

	kinds := []struct {
		name string
		kind lds.Kind
	}{
		{"halton", lds.Halton},
		{"kronecker", lds.Kronecker},
	}
	for _, k := range kinds {
		opts.Kind = k.kind
		pts, err := lds.Generate(*dim, *n, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate %s: %v\n", k.name, err)
			os.Exit(1)
		}
		r, err := lds.Assess(pts, *bins)
		if err != nil {
			fmt.Fprintf(os.Stderr, "assess %s: %v\n", k.name, err)
			os.Exit(1)
		}
		l2 := r.L2StarDisc
		if l2 < 0 {
			fmt.Printf("%-10s %12s %12.4f %12.2f %10d %10d\n",
				k.name, "(skipped)", r.MaxAbsPairCorr, r.ChiSquareMax,
				r.EdgeCount, r.OutOfRange)
			continue
		}
		fmt.Printf("%-10s %12.6f %12.4f %12.2f %10d %10d\n",
			k.name, l2, r.MaxAbsPairCorr, r.ChiSquareMax, r.EdgeCount, r.OutOfRange)
	}

	// Plain i.i.d. uniform reference for comparison.
	rng := rand.New(rand.NewSource(20260927))
	plain := make([][]float64, *n)
	for i := range plain {
		p := make([]float64, *dim)
		for d := range p {
			p[d] = rng.Float64()
		}
		plain[i] = p
	}
	r, err := lds.Assess(plain, *bins)
	if err != nil {
		fmt.Fprintf(os.Stderr, "assess plain: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%-10s %12.6f %12.4f %12.2f %10d %10d\n",
		"random", r.L2StarDisc, r.MaxAbsPairCorr, r.ChiSquareMax,
		r.EdgeCount, r.OutOfRange)
}
