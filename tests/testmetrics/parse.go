package testmetrics

import (
	"context"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// Query identifies a single Prometheus counter to extract from scrape text.
// When LabelFilter is empty, the first line matching Name is used.
// When set, only lines containing both Name and LabelFilter are matched.
type Query struct {
	Name        string
	LabelFilter string
}

// Value is a single scraped Prometheus counter value.
type Value struct {
	Val   float64
	Found bool
}

// Key returns a stable map key for a Query.
func Key(q Query) string {
	if q.LabelFilter == "" {
		return q.Name
	}
	return q.Name + "{" + q.LabelFilter + "}"
}

// Parse extracts the requested counters from raw Prometheus exposition text.
// The returned map is keyed by Key(query).
func Parse(text string, queries []Query) map[string]Value {
	out := make(map[string]Value, len(queries))
	for _, q := range queries {
		k := Key(q)
		if q.LabelFilter == "" {
			v, ok := parseCounter(text, q.Name)
			out[k] = Value{Val: v, Found: ok}
		} else {
			v, ok := parseCounterWithLabel(text, q.Name, q.LabelFilter)
			out[k] = Value{Val: v, Found: ok}
		}
	}
	return out
}

// ValuesEqual returns true when both maps have the same keys with identical Found/Val.
func ValuesEqual(a, b map[string]Value) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || va != vb {
			return false
		}
	}
	return true
}

// StableConfig configures PollUntilStable.
type StableConfig struct {
	PollInterval time.Duration
	StableRounds int
	Logf         func(string, ...any)
}

// ScrapeFunc fetches metrics for one poll iteration. Returning an error signals a
// retryable failure (e.g. pod not ready); the poll continues until the context expires.
type ScrapeFunc func(ctx context.Context) (map[string]Value, error)

// PollUntilStable polls scrapeFn until the returned values are identical across
// stableRounds consecutive successful scrapes, or the context expires.
// On timeout it returns the last successful result (assertions should catch the real issue).
func PollUntilStable(ctx context.Context, cfg StableConfig, scrapeFn ScrapeFunc) map[string]Value {
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	stableRounds := cfg.StableRounds
	if stableRounds <= 0 {
		stableRounds = 3
	}
	logf := cfg.Logf

	var prev map[string]Value
	consecutiveStable := 0

	_ = wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		cur, err := scrapeFn(ctx)
		if err != nil {
			consecutiveStable = 0
			prev = nil
			if logf != nil {
				logf("metrics poll: scrape error (will retry): %v", err)
			}
			return false, nil
		}

		if prev != nil && ValuesEqual(prev, cur) {
			consecutiveStable++
		} else {
			consecutiveStable = 1
		}

		if logf != nil && (consecutiveStable == 1 || consecutiveStable == stableRounds) {
			logf("metrics poll: stable=%d/%d values=%v", consecutiveStable, stableRounds, cur)
		}
		prev = cur
		return consecutiveStable >= stableRounds, nil
	})

	if prev == nil {
		return map[string]Value{}
	}
	return prev
}

func parseCounter(body, metricPrefix string) (float64, bool) {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, metricPrefix) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			continue
		}
		return val, true
	}
	return 0, false
}

func parseCounterWithLabel(body, metricName, labelSubstring string) (float64, bool) {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, metricName) {
			continue
		}
		if !strings.Contains(line, labelSubstring) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			continue
		}
		return val, true
	}
	return 0, false
}
