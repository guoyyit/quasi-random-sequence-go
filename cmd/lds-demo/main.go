// Command lds-demo generates low-discrepancy point sets with the two built-in
// mechanisms and prints deterministic quality metrics. Everything is local
// and reproducible: run it repeatedly to obtain identical output.
package main

import (
	"fmt"
	"math"
	"os"

	"github.com/guoyyit/quasi-random-sequence-go/lds"
)

// validatePoints fails (exit code 1) if any point touches the cube boundary,
// repeats within its own set, or overlaps a point already produced by another
// scrambled stream. The demo never prints a set that violates the interior,
// uniqueness, or cross-stream independence guarantees.
func validatePoints(name string, pts [][]float64, crossStream map[string]bool) {
	seen := make(map[string]bool, len(pts))
	for i, p := range pts {
		for _, x := range p {
			if math.IsNaN(x) || x <= 0.0 || x >= 1.0 {
				fmt.Fprintf(os.Stderr, "validate %s: boundary/NaN coordinate at point %d: %v\n", name, i, p)
				os.Exit(1)
			}
		}
		k := fmt.Sprint(p)
		if seen[k] {
			fmt.Fprintf(os.Stderr, "validate %s: duplicate point at index %d: %v\n", name, i, p)
			os.Exit(1)
		}
		seen[k] = true
		if crossStream != nil && crossStream[k] {
			fmt.Fprintf(os.Stderr, "validate %s: point %d overlaps another stream: %v\n", name, i, p)
			os.Exit(1)
		}
	}
	if crossStream != nil {
		for k := range seen {
			crossStream[k] = true
		}
	}
}

func main() {
	const (
		dim = 4
		n   = 400
	)

	cases := []struct {
		name string
		cfg  lds.Config
	}{
		{"Halton", lds.Config{Kind: lds.Halton, Dim: dim}},
		{"Halton scrambled (seed=1)", lds.Config{Kind: lds.Halton, Dim: dim, Scramble: true, Seed: 1}},
		{"Faure", lds.Config{Kind: lds.Faure, Dim: dim}},
		{"Faure scrambled (seed=1)", lds.Config{Kind: lds.Faure, Dim: dim, Scramble: true, Seed: 1}},
	}

	fmt.Printf("Generating n=%d points in %d dimensions\n\n", n, dim)
	type result struct {
		name   string
		points [][]float64
		m      lds.Metrics
	}
	var results []result
	for _, c := range cases {
		points, err := lds.Generate(c.cfg, n)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate %s: %v\n", c.name, err)
			os.Exit(1)
		}
		validatePoints(c.name, points, nil)
		m, err := lds.Assess(points)
		if err != nil {
			fmt.Fprintf(os.Stderr, "assess %s: %v\n", c.name, err)
			os.Exit(1)
		}
		results = append(results, result{name: c.name, points: points, m: m})
	}

	// Validation passed for every set: only now print anything.
	for _, r := range results {
		fmt.Printf("%-30s L2* discrepancy = %.6f  max|corr| = %.6f  mean|corr| = %.6f\n",
			r.name, r.m.L2StarDiscrepancy, r.m.MaxAbsCorrelation, r.m.MeanAbsCorrelation)
	}

	fmt.Println("\nReproducibility demo (Halton scrambled, seed 42):")
	streamPoints := make(map[string]bool)
	for _, stream := range []int{0, 1, 2} {
		cfg := lds.Config{Kind: lds.Halton, Dim: dim, Scramble: true, Seed: 42, Stream: stream}
		p, err := lds.Generate(cfg, 1)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		validatePoints(fmt.Sprintf("seed42 stream %d", stream), p, streamPoints)
		fmt.Printf("  stream %d first point = %v\n", stream, p[0])
	}
}
