package lds

import "math"

// Assess computes deterministic quality indicators for a point set:
//
//   - L2StarDiscrepancy: Warnock's closed-form L2 star discrepancy, a global
//     uniformity measure (smaller is more uniform).
//   - MaxAbsCorrelation / MeanAbsCorrelation: absolute Pearson correlation
//     between coordinate pairs, measuring inter-dimension dependence.
//
// The computation is deterministic and allocation-only: identical input
// always yields identical output, regardless of goroutine scheduling.
func Assess(points [][]float64) (Metrics, error) {
	if len(points) == 0 {
		return Metrics{}, ErrInvalidPoints
	}
	dim := len(points[0])
	if dim <= 0 {
		return Metrics{}, ErrInvalidPoints
	}
	n := len(points)
	if n > maxAssessSamples {
		return Metrics{}, ErrSamplesTooLarge
	}
	for _, p := range points {
		if len(p) != dim {
			return Metrics{}, ErrInvalidPoints
		}
		for _, x := range p {
			if math.IsNaN(x) || math.IsInf(x, 0) || x < 0.0 || x > 1.0 {
				return Metrics{}, ErrInvalidPoints
			}
		}
	}

	m := Metrics{N: n, Dim: dim}
	m.L2StarDiscrepancy = l2StarDiscrepancy(points)
	m.MaxAbsCorrelation, m.MeanAbsCorrelation = coordinateCorrelations(points)
	return m, nil
}

// l2StarDiscrepancy evaluates Warnock's formula:
//
//	D2^2 = (1/3)^s
//	  - (2/n) sum_i prod_d (1 - x_{i,d}^2)/2
//	  + (1/n^2) sum_i sum_k prod_d (1 - max(x_{i,d}, x_{k,d})).
func l2StarDiscrepancy(points [][]float64) float64 {
	n := len(points)
	s := len(points[0])

	termSingle := 0.0
	for i := 0; i < n; i++ {
		prod := 1.0
		for d := 0; d < s; d++ {
			x := points[i][d]
			prod *= (1.0 - x*x) * 0.5
		}
		termSingle += prod
	}

	termDouble := 0.0
	for i := 0; i < n; i++ {
		for k := 0; k < n; k++ {
			prod := 1.0
			for d := 0; d < s; d++ {
				a, b := points[i][d], points[k][d]
				if b > a {
					a = b
				}
				prod *= 1.0 - a
			}
			termDouble += prod
		}
	}

	fn := float64(n)
	d2 := math.Pow(1.0/3.0, float64(s))
	d2 -= (2.0 / fn) * termSingle
	d2 += (1.0 / (fn * fn)) * termDouble
	if d2 < 0 { // guard against tiny negative round-off
		d2 = 0
	}
	return math.Sqrt(d2)
}

// coordinateCorrelations returns the maximum and mean absolute off-diagonal
// Pearson correlation between coordinate columns.
func coordinateCorrelations(points [][]float64) (maxAbs, meanAbs float64) {
	n := len(points)
	dim := len(points[0])
	if dim < 2 {
		return 0, 0
	}

	means := make([]float64, dim)
	for _, p := range points {
		for d := 0; d < dim; d++ {
			means[d] += p[d]
		}
	}
	for d := range means {
		means[d] /= float64(n)
	}
	// Centered coordinate columns.
	centered := make([][]float64, dim)
	norms := make([]float64, dim)
	for d := 0; d < dim; d++ {
		col := make([]float64, n)
		var norm float64
		for i, p := range points {
			c := p[d] - means[d]
			col[i] = c
			norm += c * c
		}
		centered[d] = col
		norms[d] = norm
	}

	pairs := 0
	for a := 0; a < dim; a++ {
		for b := a + 1; b < dim; b++ {
			var corr float64
			if norms[a] > 0 && norms[b] > 0 {
				dot := 0.0
				ca, cb := centered[a], centered[b]
				for i := 0; i < n; i++ {
					dot += ca[i] * cb[i]
				}
				corr = dot / math.Sqrt(norms[a]*norms[b])
				if corr > 1 {
					corr = 1
				} else if corr < -1 {
					corr = -1
				}
			}
			abs := math.Abs(corr)
			if abs > maxAbs {
				maxAbs = abs
			}
			meanAbs += abs
			pairs++
		}
	}
	meanAbs /= float64(pairs)
	return maxAbs, meanAbs
}
