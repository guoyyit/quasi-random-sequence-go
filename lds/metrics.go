package lds

import (
	"errors"
	"math"
)

// QualityReport collects deterministic quality indicators for a point set in
// the unit cube. All indicators are pure functions of the input points: the
// same points always produce the same report, so reports can be compared
// across sequence kinds and seeds.
type QualityReport struct {
	N   int
	Dim int

	// L2StarDisc is the Warnock–Hyde L2 star discrepancy, a measure of the
	// deviation of the empirical distribution from uniformity over anchored
	// boxes [0,x). Smaller is better; a plain i.i.d.-uniform sample of size n
	// is expected near ~0.4/sqrt(n). It is set to -1 when n exceeds
	// L2MaxN and the O(n^2) computation was skipped.
	L2StarDisc float64

	// MaxAbsPairCorr is the largest absolute Pearson correlation across all
	// pairs of dimensions (0 is ideal).
	MaxAbsPairCorr float64

	// PairCorrs holds the upper-triangular per-pair correlations in the
	// order (0,1),(0,2),...,(0,d-1),(1,2),...
	PairCorrs []float64

	// ChiSquare holds, per dimension, the Pearson chi-square statistic of
	// counts in bins equal-probability cells against the uniform expectation.
	ChiSquare []float64

	// ChiSquareMax is max(ChiSquare).
	ChiSquareMax float64

	// EdgeCount is the number of points touching a cube boundary
	// (any coordinate equal to 0 or 1).
	EdgeCount int

	// OutOfRange is the number of points with a coordinate outside [0,1].
	// When positive the report is returned with a non-nil error.
	OutOfRange int
}

// L2MaxN is the largest point count for which the quadratic L2 star
// discrepancy is computed.
const L2MaxN = 2000

// DefaultBins is used by Assess when bins == 0.
const DefaultBins = 16

// Assess computes the full quality report. bins selects the uniform grid for
// the chi-square statistic (bins == 0 selects DefaultBins; otherwise
// 2 <= bins <= len(points) is required).
func Assess(points [][]float64, bins int) (*QualityReport, error) {
	n, dim, outOfRange, edge, err := checkPoints(points)
	if err != nil {
		return nil, err
	}
	if bins == 0 {
		bins = DefaultBins
	}
	if bins < 2 || bins > n {
		return nil, errors.Join(ErrInvalidSamples, errors.New("lds: require 2 <= bins <= n"))
	}

	r := &QualityReport{
		N:          n,
		Dim:        dim,
		OutOfRange: outOfRange,
		EdgeCount:  edge,
	}

	if outOfRange > 0 {
		// Return the diagnostics plus an explicit error; never silently
		// evaluate discrepancy metrics on illegal coordinates.
		return r, errors.Join(ErrInvalidDimension, errors.New("lds: points contain coordinates outside [0,1]"))
	}

	if n <= L2MaxN {
		r.L2StarDisc = l2StarDiscrepancy(points, n, dim)
	} else {
		r.L2StarDisc = -1
	}

	r.PairCorrs, r.MaxAbsPairCorr = pairCorrelations(points, n, dim)
	r.ChiSquare, r.ChiSquareMax = uniformChiSquare(points, n, dim, bins)
	return r, nil
}

// L2StarDiscrepancy computes the Warnock–Hyde L2 star discrepancy, which can
// be evaluated in O(n^2*dim) without a search.
func L2StarDiscrepancy(points [][]float64) (float64, error) {
	n, dim, outOfRange, _, err := checkPoints(points)
	if err != nil {
		return 0, err
	}
	if outOfRange > 0 {
		return 0, errors.Join(ErrInvalidDimension, errors.New("lds: points contain coordinates outside [0,1]"))
	}
	if n > L2MaxN {
		return 0, errors.Join(ErrTooLarge, errors.New("lds: L2 discrepancy requires n <= L2MaxN"))
	}
	return l2StarDiscrepancy(points, n, dim), nil
}

// PairCorrelations returns the upper-triangular Pearson correlations and the
// maximum absolute value.
func PairCorrelations(points [][]float64) ([]float64, float64, error) {
	n, dim, _, _, err := checkPoints(points)
	if err != nil {
		return nil, 0, err
	}
	c, m := pairCorrelations(points, n, dim)
	return c, m, nil
}

// UniformChiSquare computes per-dimension Pearson chi-square statistics
// against bins equal-probability cells and returns them and their maximum.
func UniformChiSquare(points [][]float64, bins int) ([]float64, float64, error) {
	n, dim, outOfRange, _, err := checkPoints(points)
	if err != nil {
		return nil, 0, err
	}
	if bins < 2 || bins > n {
		return nil, 0, errors.Join(ErrInvalidSamples, errors.New("lds: require 2 <= bins <= n"))
	}
	if outOfRange > 0 {
		return nil, 0, errors.Join(ErrInvalidDimension, errors.New("lds: points contain coordinates outside [0,1]"))
	}
	c, m := uniformChiSquare(points, n, dim, bins)
	return c, m, nil
}

// checkPoints validates the matrix shape and counts boundary/out-of-range
// coordinates.
func checkPoints(points [][]float64) (n, dim, outOfRange, edge int, err error) {
	if len(points) == 0 {
		return 0, 0, 0, 0, errors.Join(ErrInvalidSamples, errors.New("lds: no points to assess"))
	}
	dim = len(points[0])
	if dim == 0 {
		return 0, 0, 0, 0, errors.Join(ErrInvalidDimension, errors.New("lds: points have zero dimension"))
	}
	for _, p := range points {
		if len(p) != dim {
			return 0, 0, 0, 0, errors.Join(ErrInvalidDimension, errors.New("lds: ragged point matrix"))
		}
	}
	n = len(points)
	for _, p := range points {
		bad := false
		onEdge := false
		for _, x := range p {
			if math.IsNaN(x) || math.IsInf(x, 0) {
				return 0, 0, 0, 0, errors.Join(ErrInvalidDimension, errors.New("lds: non-finite coordinate"))
			}
			if x < 0 || x > 1 {
				bad = true
			}
			if x == 0 || x == 1 {
				onEdge = true
			}
		}
		if bad {
			outOfRange++
		}
		if onEdge {
			edge++
		}
	}
	return n, dim, outOfRange, edge, nil
}

// l2StarDiscrepancy evaluates
//
//	D2^2 = 1/n^2 sum_{i,j} prod_d (1 - max(x_id,x_jd))
//	       - 2^(1-d)/n sum_i prod_d (1 - x_id^2)
//	       + 3^(-d).
func l2StarDiscrepancy(points [][]float64, n, dim int) float64 {
	sumPairs := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			prod := 1.0
			for d := 0; d < dim; d++ {
				m := points[i][d]
				if points[j][d] > m {
					m = points[j][d]
				}
				prod *= 1 - m
			}
			sumPairs += prod
		}
	}
	sumSelf := 0.0
	for i := 0; i < n; i++ {
		prod := 1.0
		for d := 0; d < dim; d++ {
			prod *= 1 - points[i][d]*points[i][d]
		}
		sumSelf += prod
	}
	volumeTerm := math.Pow(1.0/3.0, float64(dim))
	fn := float64(n)
	v := sumPairs/(fn*fn) - math.Ldexp(1.0, 1-dim)*sumSelf/fn + volumeTerm
	if v < 0 {
		v = 0 // guard against tiny negative rounding noise
	}
	return math.Sqrt(v)
}

// pairCorrelations computes all upper-triangular Pearson correlations.
func pairCorrelations(points [][]float64, n, dim int) ([]float64, float64) {
	mean := make([]float64, dim)
	for _, p := range points {
		for d := 0; d < dim; d++ {
			mean[d] += p[d]
		}
	}
	fn := float64(n)
	for d := range mean {
		mean[d] /= fn
	}
	cov := make([]float64, dim*dim)
	vari := make([]float64, dim)
	for _, p := range points {
		for a := 0; a < dim; a++ {
			da := p[a] - mean[a]
			vari[a] += da * da
			for b := a + 1; b < dim; b++ {
				db := p[b] - mean[b]
				cov[a*dim+b] += da * db
			}
		}
	}
	var out []float64
	maxAbs := 0.0
	for a := 0; a < dim; a++ {
		for b := a + 1; b < dim; b++ {
			c := 0.0
			if vari[a] > 0 && vari[b] > 0 {
				c = cov[a*dim+b] / math.Sqrt(vari[a]*vari[b])
			}
			out = append(out, c)
			if ac := math.Abs(c); ac > maxAbs {
				maxAbs = ac
			}
		}
	}
	return out, maxAbs
}

// uniformChiSquare computes per-dimension Pearson chi-square against a uniform
// grid of bins cells in [0,1).
func uniformChiSquare(points [][]float64, n, dim, bins int) ([]float64, float64) {
	expected := float64(n) / float64(bins)
	out := make([]float64, dim)
	maxVal := 0.0
	for d := 0; d < dim; d++ {
		counts := make([]int, bins)
		for _, p := range points {
			b := int(p[d] * float64(bins))
			if b >= bins {
				b = bins - 1 // coordinate exactly 1
			}
			if b < 0 {
				b = 0
			}
			counts[b]++
		}
		chi := 0.0
		for _, c := range counts {
			diff := float64(c) - expected
			chi += diff * diff / expected
		}
		out[d] = chi
		if chi > maxVal {
			maxVal = chi
		}
	}
	return out, maxVal
}
