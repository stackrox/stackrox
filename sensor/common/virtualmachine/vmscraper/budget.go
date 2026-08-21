package vmscraper

import (
	"time"
)

// startBudget is max(1, ceil(n × tick / window)) for positive n, tick, and
// window. Zero otherwise so a missing window does not start a scrape.
func startBudget(n int, tick, window time.Duration) int {
	if n <= 0 || tick <= 0 || window <= 0 {
		return 0
	}
	num := int64(n) * int64(tick)
	den := int64(window)
	return max(1, int((num+den-1)/den))
}
