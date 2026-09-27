package lds

import (
	"math"
	"testing"
)

func TestVanDerCorputKnownValues(t *testing.T) {
	// Classical (unscrambled) base-2 radical inverses for i=1..8.
	want := []float64{0.5, 0.25, 0.75, 0.125, 0.625, 0.375, 0.875, 0.0625}
	for i, w := range want {
		if got := radicalInverse(int64(i+1), 2, nil); got != w {
			t.Fatalf("i=%d got %.4f want %.4f", i+1, got, w)
		}
	}
}

func TestHaltonKnownFirstPoint(t *testing.T) {
	// First Halton point (index 1) in 3D: (1/2, 1/3, 1/5).
	pts, err := Generate(3, 1, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{0.5, 1.0 / 3.0, 0.2}
	for d := range want {
		if pts[0][d] != want[d] {
			t.Fatalf("coord %d = %.17g, want %.17g", d, pts[0][d], want[d])
		}
	}
}

func TestKroneckerKnownValues(t *testing.T) {
	// Richtmyer one-dimensional sequence {i*sqrt(2)} for i=1..5.
	alpha := math.Sqrt(2) - 1
	pts, err := Generate(1, 5, Options{Kind: Kronecker})
	if err != nil {
		t.Fatal(err)
	}
	wantVals := []float64{}
	for i := 1; i <= 5; i++ {
		_, f := math.Modf(float64(i) * math.Sqrt(2))
		wantVals = append(wantVals, f)
	}
	const tol = 1e-15
	for i, w := range wantVals {
		if d := pts[i][0] - w; d > tol || d < -tol {
			t.Fatalf("i=%d got %.17g want %.17g", i+1, pts[i][0], w)
		}
	}
	// First coordinate is sqrt(2)-1 ~ 0.4142.
	if d := pts[0][0] - alpha; d > tol || d < -tol {
		t.Fatalf("first = %.17g want %.17g", pts[0][0], alpha)
	}
}

func TestNoDuplicatePoints(t *testing.T) {
	for _, k := range kinds {
		pts, err := Generate(3, 300, Options{Kind: k, Scramble: true, Seed: 99})
		if err != nil {
			t.Fatal(err)
		}
		seen := make(map[[3]float64]struct{}, len(pts))
		for _, p := range pts {
			key := [3]float64{p[0], p[1], p[2]}
			if _, dup := seen[key]; dup {
				t.Fatalf("%s: duplicate point %v", k, key)
			}
			seen[key] = struct{}{}
		}
	}
}

func TestScramblePreservesInteriors(t *testing.T) {
	for _, k := range kinds {
		for d := 1; d <= 10; d++ {
			pts, err := Generate(d, 100, Options{Kind: k, Scramble: true, Seed: 4242})
			if err != nil {
				t.Fatal(err)
			}
			r, err := Assess(pts, 10)
			if err != nil {
				t.Fatal(err)
			}
			if r.OutOfRange != 0 {
				t.Fatalf("%s d=%d: %d points out of range", k, d, r.OutOfRange)
			}
		}
	}
}
