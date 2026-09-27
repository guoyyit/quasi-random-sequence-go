package lds

// primes returns the first n primes, computed once at package init.
// The 1000th prime is 7919, so a sieve to a fixed bound is sufficient and
// fully deterministic (no network or OS randomness involved).
var firstPrimes = sievePrimes(1000)

// sievePrimes computes the first count primes with the Eratosthenes sieve.
func sievePrimes(count int) []int {
	if count <= 0 {
		return nil
	}
	// Upper bound via the elementary estimate p_n < n*(log n + log log n)+10,
	// with a generous floor for small n.
	limit := 100
	for primesBelow(limit) < count {
		limit *= 2
	}
	composite := make([]bool, limit+1)
	primes := make([]int, 0, count)
	for i := 2; i <= limit && len(primes) < count; i++ {
		if composite[i] {
			continue
		}
		primes = append(primes, i)
		for j := i * i; j <= limit; j += i {
			composite[j] = true
		}
	}
	return primes
}

// primesBelow counts primes up to and including limit via a fresh sieve.
func primesBelow(limit int) int {
	if limit < 2 {
		return 0
	}
	composite := make([]bool, limit+1)
	count := 0
	for i := 2; i <= limit; i++ {
		if composite[i] {
			continue
		}
		count++
		for j := i * i; j <= limit; j += i {
			composite[j] = true
		}
	}
	return count
}

// primeAtLeast returns the smallest prime >= b, searching the precomputed
// table and extending with trial division if the table is exhausted.
func primeAtLeast(b int) int {
	for _, p := range firstPrimes {
		if p >= b {
			return p
		}
	}
	for x := b | 1; ; x += 2 {
		if isPrime(x) {
			return x
		}
	}
}

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	for p := 2; p*p <= n; p++ {
		if n%p == 0 {
			return false
		}
	}
	return true
}
