package lds

import "math/big"

// haltonGen implements the Halton sequence: dimension d uses the radical
// inverse in the d-th prime base. Optionally each digit is permuted with a
// Kocis-Owen style permutation, which preserves the low-discrepancy property.
type haltonGen struct {
	cfg   Config
	perms *permTables
	bases []int
}

func newHalton(cfg Config) (*haltonGen, error) {
	bases := make([]int, cfg.Dim)
	copy(bases, firstPrimes[:cfg.Dim])
	g := &haltonGen{cfg: cfg, bases: bases}
	if cfg.Scramble {
		g.perms = newPermTables(cfg.Seed, cfg.Stream)
	}
	return g, nil
}

// pointAt returns the i-th point, i >= 1 (the all-zero origin index 0 is
// skipped so generated point sets never contain the degenerate origin).
func (g *haltonGen) pointAt(i int) []float64 {
	point := make([]float64, g.cfg.Dim)
	for d, base := range g.bases {
		x := i
		// Collect base-b digits, least significant first.
		var digits []int
		for x > 0 {
			digits = append(digits, x%base)
			x /= base
		}
		if g.perms != nil {
			// Full Owen scramble including the infinite tail: the tail keeps
			// the coordinate strictly inside (0,1) and prevents collisions.
			point[d] = scrambledValue(g.perms, base, d, digits)
		} else {
			// Unscrambled: exact rational num/base^k. Dividing integer
			// numerator by integer denominator once gives the correctly
			// rounded float64, identical between incremental and batch.
			var num, den int64 = 0, 1
			for _, digit := range digits {
				num = num*int64(base) + int64(digit)
				den *= int64(base)
			}
			point[d] = float64(num) / float64(den)
		}
	}
	return point
}

// faureGen implements the Faure digital net. With s dimensions it chooses the
// smallest prime b >= s (b = 2 when s = 1), expands index k in base b and
// multiplies the digit vector by powers of Pascal's matrix modulo b. Each
// power defines one dimension. Scrambling permutes the resulting digits.
type faureGen struct {
	cfg   Config
	perms *permTables
	base  int
	k     int // fixed number of base-b digits (precision, independent of index)
	// mats[d] is P^d as a k*k row-major upper-triangular matrix modulo b,
	// with entry (i, r) = C(r, i) * d^(r-i) mod b for r >= i.
	mats [][]int
}

func newFaure(cfg Config) (*faureGen, error) {
	base := 2
	if cfg.Dim > 1 {
		base = primeAtLeast(cfg.Dim)
	}
	// Fixed precision covering every supported index. Using an index-
	// independent precision makes point i identical whether produced via a
	// stream or a batch of any size n >= i.
	k := digitsInBase(maxGenerateSamples, base)
	g := &faureGen{base: base, k: k}
	g.cfg = cfg
	g.mats = make([][]int, cfg.Dim)
	for d := 0; d < cfg.Dim; d++ {
		m := make([]int, k*k)
		for r := 0; r < k; r++ {
			for i := 0; i <= r; i++ {
				v := new(big.Int).Mod(binom(r, i), big.NewInt(int64(base)))
				pow := modPow(d, r-i, base)
				m[i*k+r] = int(v.Int64()) * pow % base
			}
		}
		g.mats[d] = m
	}
	if cfg.Scramble {
		g.perms = newPermTables(cfg.Seed, cfg.Stream)
	}
	return g, nil
}

// pointAt returns the i-th point, i >= 1.
func (g *faureGen) pointAt(i int) []float64 {
	b := g.base
	k := g.k
	// Base-b digits, least significant first.
	a := make([]int, k)
	x := i
	for pos := 0; pos < k && x > 0; pos++ {
		a[pos] = x % b
		x /= b
	}
	point := make([]float64, g.cfg.Dim)
	for d := 0; d < g.cfg.Dim; d++ {
		m := g.mats[d]
		// Output digits y = M a mod b, least significant first.
		y := make([]int, k)
		for row := 0; row < k; row++ {
			v := 0
			baseRow := row * k
			for col := row; col < k; col++ {
				v += m[baseRow+col] * a[col]
			}
			y[row] = v % b
		}
		if g.perms != nil {
			point[d] = scrambledValue(g.perms, b, d, y)
		} else {
			// Unscrambled: exact rational num/b^k.
			var num, den int64 = 0, 1
			for _, v := range y {
				num = num*int64(b) + int64(v)
				den *= int64(b)
			}
			point[d] = float64(num) / float64(den)
		}
	}
	return point
}

// modPow computes base^exp mod m.
func modPow(base, exp, m int) int {
	if m == 1 {
		return 0
	}
	result := 1 % m
	b := base % m
	for exp > 0 {
		if exp&1 == 1 {
			result = result * b % m
		}
		b = b * b % m
		exp >>= 1
	}
	return result
}

// digitsInBase returns the number of base-b digits of n (n >= 1 -> >= 1).
func digitsInBase(n, b int) int {
	if n <= 0 {
		return 0
	}
	k := 0
	for n > 0 {
		n /= b
		k++
	}
	return k
}

// binom computes the binomial coefficient C(n, k) with arbitrary precision so
// that parameter combinations prone to machine-word overflow never silently
// wrap around.
func binom(n, k int) *big.Int {
	if k < 0 || k > n {
		return big.NewInt(0)
	}
	if k == 0 || k == n {
		return big.NewInt(1)
	}
	if k > n-k {
		k = n - k
	}
	c := big.NewInt(1)
	for i := 0; i < k; i++ {
		c.Mul(c, big.NewInt(int64(n-i)))
		c.Div(c, big.NewInt(int64(i+1)))
	}
	return c
}
