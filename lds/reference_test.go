package lds

import "testing"

// Known first points of the classical Halton sequence in bases 2,3,5.
func TestHaltonReference(t *testing.T) {
	cfg := Config{Kind: Halton, Dim: 3}
	pts, err := Generate(cfg, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]float64{
		{0.5, 1.0 / 3.0, 0.2},
		{0.25, 2.0 / 3.0, 0.4},
		{0.75, 1.0 / 9.0, 0.6},
		{0.125, 4.0 / 9.0, 0.8},
	}
	if !pointSetsEqual(pts, want) {
		t.Fatalf("halton reference mismatch:\n got %v\nwant %v", pts, want)
	}
}

// Known first points of the classical Faure sequence for s=3 (base 3).
// Point i uses base-3 digits and Pascal matrices P^j.
func TestFaureReference(t *testing.T) {
	cfg := Config{Kind: Faure, Dim: 3}
	pts, err := Generate(cfg, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]float64{
		{1.0 / 3.0, 1.0 / 3.0, 1.0 / 3.0},
		{2.0 / 3.0, 2.0 / 3.0, 2.0 / 3.0},
		{1.0 / 9.0, 4.0 / 9.0, 7.0 / 9.0},
	}
	if !pointSetsEqual(pts, want) {
		t.Fatalf("faure reference mismatch:\n got %v\nwant %v", pts, want)
	}
}
