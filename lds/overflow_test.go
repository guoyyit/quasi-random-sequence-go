package lds

import (
	"errors"
	"testing"
)

func TestStartIndexOverflow(t *testing.T) {
	// One point whose absolute generator index (start+k+1) equals the limit.
	pts, err := Generate(1, 1, Options{StartIndex: maxAbsoluteIndex - 2})
	if err != nil {
		t.Fatalf("boundary StartIndex should be accepted: %v", err)
	}
	if len(pts) != 1 || pts[0][0] <= 0 || pts[0][0] >= 1 {
		t.Fatalf("bad boundary point: %v", pts)
	}

	// One point beyond the limit must be rejected, not panic.
	if _, err := Generate(1, 1, Options{StartIndex: maxAbsoluteIndex - 1}); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("got %v, want ErrTooLarge", err)
	}
	// Two points straddling the limit.
	if _, err := Generate(1, 2, Options{StartIndex: maxAbsoluteIndex - 2}); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("got %v, want ErrTooLarge", err)
	}
}

func TestStreamOverflow(t *testing.T) {
	s, err := NewSequence(1, Options{StartIndex: maxAbsoluteIndex - 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Next(); err != nil {
		t.Fatalf("first Next at boundary: %v", err)
	}
	if _, err := s.Next(); err != nil {
		t.Fatalf("second Next (absolute index = limit) should succeed: %v", err)
	}
	if _, err := s.Next(); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("third Next: got %v, want ErrTooLarge", err)
	}

	s2, _ := NewSequence(1, Options{StartIndex: maxAbsoluteIndex - 2})
	if _, err := s2.NextN(2); err != nil {
		t.Fatalf("NextN up to limit: %v", err)
	}
	if _, err := s2.NextN(1); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("NextN beyond limit: got %v, want ErrTooLarge", err)
	}
}
