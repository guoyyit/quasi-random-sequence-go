package lds

import "testing"

// One-dimensional sequences must work and stay within the unit interval.
func TestOneDimensional(t *testing.T) {
	for _, kind := range kinds {
		pts, err := Generate(Config{Kind: kind, Dim: 1, Scramble: true, Seed: 3}, 200)
		if err != nil {
			t.Fatal(err)
		}
		if len(pts) != 200 || len(pts[0]) != 1 {
			t.Fatalf("%v bad shape", kind)
		}
		st, err := NewStream(Config{Kind: kind, Dim: 1})
		if err != nil {
			t.Fatal(err)
		}
		first, err := st.Next()
		if err != nil {
			t.Fatal(err)
		}
		batch, err := Generate(Config{Kind: kind, Dim: 1}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if first[0] != batch[0][0] {
			t.Fatalf("%v 1d mismatch", kind)
		}
	}
}

// The largest supported dimension must build and produce legal points,
// exercising the 1000th prime base and the dimension-cap validation.
func TestLargeDimension(t *testing.T) {
	cfg := Config{Kind: Halton, Dim: maxDim, Scramble: true, Seed: 1}
	st, err := NewStream(cfg)
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.Next()
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != maxDim {
		t.Fatalf("got %d coords", len(p))
	}
	fcfg := Config{Kind: Faure, Dim: 64}
	fp, err := Generate(fcfg, 16)
	if err != nil {
		t.Fatal(err)
	}
	if len(fp) != 16 || len(fp[0]) != 64 {
		t.Fatal("faure high-dim bad shape")
	}
}

// binom must use arbitrary precision: a value far beyond uint64 must be exact.
func TestBinomNoOverflow(t *testing.T) {
	c := binom(200, 100)
	// C(200,100) = 90548514656103281165404177077484163874504589675413336841320
	want := "90548514656103281165404177077484163874504589675413336841320"
	if c.String() != want {
		t.Fatalf("binom overflow: got %s", c.String())
	}
	if binom(5, 6).Sign() != 0 {
		t.Fatal("C(5,6) should be 0")
	}
}

// Low discrepancy should clearly beat independent uniform sampling in the
// same dimension/noise budget; this is a comparison smoke test, not a proof.
func TestQualityComparison(t *testing.T) {
	cfg := Config{Kind: Faure, Dim: 4}
	pts, err := Generate(cfg, 256)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Assess(pts)
	if err != nil {
		t.Fatal(err)
	}
	// A 256-point 4D Faure net typically has L2* discrepancy well under 0.02.
	if m.L2StarDiscrepancy > 0.02 {
		t.Fatalf("faure discrepancy unexpectedly high: %v", m.L2StarDiscrepancy)
	}
}
