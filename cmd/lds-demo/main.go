// Command lds-demo generates low-discrepancy point sets with the two built-in
// mechanisms and prints deterministic quality metrics. Everything is local
// and reproducible: run it repeatedly to obtain identical output.
package main

import (
	"fmt"
	"os"

	"github.com/guoyyit/quasi-random-sequence-go/lds"
)

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
	for _, c := range cases {
		points, err := lds.Generate(c.cfg, n)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate %s: %v\n", c.name, err)
			os.Exit(1)
		}
		m, err := lds.Assess(points)
		if err != nil {
			fmt.Fprintf(os.Stderr, "assess %s: %v\n", c.name, err)
			os.Exit(1)
		}
		fmt.Printf("%-30s L2* discrepancy = %.6f  max|corr| = %.6f  mean|corr| = %.6f\n",
			c.name, m.L2StarDiscrepancy, m.MaxAbsCorrelation, m.MeanAbsCorrelation)
	}

	fmt.Println("\nReproducibility demo (Halton scrambled, seed 42):")
	for _, stream := range []int{0, 1, 2} {
		cfg := lds.Config{Kind: lds.Halton, Dim: dim, Scramble: true, Seed: 42, Stream: stream}
		p, err := lds.Generate(cfg, 1)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("  stream %d first point = %v\n", stream, p[0])
	}
}
