package lds

import (
	"errors"
	"math"
	"math/rand"
	"testing"
)

func TestAssessDeterministic(t *testing.T) {
	pts, err := Generate(4, 500, Options{Scramble: true, Seed: 123})
	if err != nil {
		t.Fatal(err)
	}
	a, err := Assess(pts, 16)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Assess(pts, 16)
	if err != nil {
		t.Fatal(err)
	}
	if a.L2StarDisc != b.L2StarDisc ||
		a.MaxAbsPairCorr != b.MaxAbsPairCorr ||
		a.ChiSquareMax != b.ChiSquareMax {
		t.Fatal("Assess is not deterministic for identical input")
	}
	if a.L2StarDisc <= 0 {
		t.Fatalf("L2 discrepancy must be positive, got %v", a.L2StarDisc)
	}
	t.Logf("scrambled Halton n=500 d=4: L2*=%.5f maxCorr=%.4f chiMax=%.2f",
		a.L2StarDisc, a.MaxAbsPairCorr, a.ChiSquareMax)
}

func TestLDSBeatsPlainRandom(t *testing.T) {
	const n, dim, bins = 1000, 3, 20
	rng := rand.New(rand.NewSource(20260927))
	plain := make([][]float64, n)
	for i := range plain {
		plain[i] = []float64{rng.Float64(), rng.Float64(), rng.Float64()}
	}
	wins := 0
	for _, k := range kinds {
		ldsPts, err := Generate(dim, n, Options{Kind: k})
		if err != nil {
			t.Fatal(err)
		}
		ldsReport, err := Assess(ldsPts, bins)
		if err != nil {
			t.Fatal(err)
		}
		plainReport, err := Assess(plain, bins)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%-9s L2*=%.5f corr=%.4f chiMax=%.2f | plain L2*=%.5f corr=%.4f chiMax=%.2f",
			k, ldsReport.L2StarDisc, ldsReport.MaxAbsPairCorr, ldsReport.ChiSquareMax,
			plainReport.L2StarDisc, plainReport.MaxAbsPairCorr, plainReport.ChiSquareMax)
		if ldsReport.L2StarDisc < plainReport.L2StarDisc {
			wins++
		}
		if ldsReport.MaxAbsPairCorr > 0.3 {
			t.Fatalf("%s suspiciously high correlation: %v", k, ldsReport.MaxAbsPairCorr)
		}
	}
	if wins == 0 {
		t.Fatal("no LDS kind improved L2 discrepancy over i.i.d. uniform")
	}
}

func TestPairCorrelationOrthogonalGrid(t *testing.T) {
	// Independent linear ramps -> correlation ~0.
	pts := make([][]float64, 200)
	for i := range pts {
		x := float64(i) / 200
		pts[i] = []float64{x, 1 - x, (x*0.3 + 0.1)}
	}
	corrs, maxAbs, err := PairCorrelations(pts)
	if err != nil {
		t.Fatal(err)
	}
	// Pair (0,1) must be ~-1.
	if math.Abs(corrs[0]+1) > 1e-9 {
		t.Fatalf("corr(x,1-x) = %v, want -1", corrs[0])
	}
	if math.Abs(maxAbs-1) > 1e-9 {
		t.Fatalf("maxAbs = %v, want 1", maxAbs)
	}
}

func TestAssessOutOfRangeAndEdges(t *testing.T) {
	good, _ := Generate(2, 50, Options{})
	// LDS points are interior: zero edges expected for Kronecker/Halton from
	// index 1.
	r, err := Assess(good, 10)
	if err != nil {
		t.Fatal(err)
	}
	if r.OutOfRange != 0 {
		t.Fatalf("unexpected out-of-range count %d", r.OutOfRange)
	}

	bad := make([][]float64, 4)
	bad[0] = []float64{0.5, 0.5}
	bad[1] = []float64{1.2, 0.5} // out of range
	bad[2] = []float64{-0.1, 0}  // out of range + edge
	bad[3] = []float64{0.5, 1}   // edge
	_, err = Assess(bad, 2)
	if !errors.Is(err, ErrInvalidDimension) {
		t.Fatalf("want ErrInvalidDimension, got %v", err)
	}
}

func TestAssessShapeErrors(t *testing.T) {
	if _, err := Assess(nil, 5); !errors.Is(err, ErrInvalidSamples) {
		t.Fatalf("nil points: %v", err)
	}
	if _, err := Assess([][]float64{{}}, 5); !errors.Is(err, ErrInvalidDimension) {
		t.Fatalf("zero-dim point: %v", err)
	}
	ragged := [][]float64{{0.1, 0.2}, {0.3}}
	if _, err := Assess(ragged, 5); !errors.Is(err, ErrInvalidDimension) {
		t.Fatalf("ragged: %v", err)
	}
	ok := [][]float64{{0.5}, {0.2}, {0.9}}
	if _, err := Assess(ok, 1); !errors.Is(err, ErrInvalidSamples) {
		t.Fatalf("bins<2: %v", err)
	}
	if _, err := Assess(ok, 10); !errors.Is(err, ErrInvalidSamples) {
		t.Fatalf("bins>n: %v", err)
	}
}

func TestL2DiscrepancySkipsLargeN(t *testing.T) {
	pts, err := Generate(2, L2MaxN+1, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := L2StarDiscrepancy(pts); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("got %v", err)
	}
	r, err := Assess(pts, 16)
	if err != nil {
		t.Fatal(err)
	}
	if r.L2StarDisc != -1 {
		t.Fatalf("expected -1 sentinel, got %v", r.L2StarDisc)
	}
}

func TestNonFiniteCoordinatesRejected(t *testing.T) {
	pts := [][]float64{{0.5, math.NaN()}, {0.2, 0.3}}
	if _, err := Assess(pts, 2); err == nil {
		t.Fatal("NaN coordinate should be rejected")
	}
}
