package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

// LoadScorecard reads a scorecard JSON file produced by sensor-bench.
func LoadScorecard(path string) (*Scorecard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "read scorecard %q", path)
	}
	var sc Scorecard
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, errors.Wrapf(err, "parse scorecard %q", path)
	}
	if sc.Schema != ScorecardSchema {
		return nil, errors.Errorf("scorecard %q: unsupported schema %q (want %s)", path, sc.Schema, ScorecardSchema)
	}
	return &sc, nil
}

// CompareScorecards formats a markdown comparison table for candidate vs baseline.
// candidate is typically the newer run (e.g. PR head); baseline is the reference (e.g. merge-base).
func CompareScorecards(candidate, baseline *Scorecard) (string, error) {
	if candidate == nil || baseline == nil {
		return "", errors.New("candidate and baseline scorecards are required")
	}
	if err := validateComparable(candidate, baseline); err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprint(&b, "## Sensor benchmark comparison\n\n")
	fmt.Fprint(&b, "| | Baseline | Candidate |\n")
	fmt.Fprint(&b, "|---|---|---|\n")
	fmt.Fprintf(&b, "| Scenario | `%s` v%s (%s) | same |\n",
		baseline.Scenario.ID, baseline.Scenario.Version, baseline.Scenario.Maturity)
	fmt.Fprintf(&b, "| Success | %t | %t |\n", baseline.Run.Success, candidate.Run.Success)
	if baseline.Run.GitSHA != "" || candidate.Run.GitSHA != "" {
		fmt.Fprintf(&b, "| Git SHA | `%s` | `%s` |\n", baseline.Run.GitSHA, candidate.Run.GitSHA)
	}
	fmt.Fprintf(&b, "| Measure (s) | %.0f | %.0f |\n", baseline.Run.MeasureSec, candidate.Run.MeasureSec)
	fmt.Fprintf(&b, "| Sync wait (s) | %.1f | %.1f |\n\n", baseline.Run.SyncWaitSec, candidate.Run.SyncWaitSec)

	fmt.Fprint(&b, "### Steady metrics\n\n")
	fmt.Fprint(&b, "| Metric | Baseline | Candidate | Δ% |\n")
	fmt.Fprint(&b, "|--------|----------|-----------|-----|\n")

	baseByID := metricsByID(baseline)
	candByID := metricsByID(candidate)
	for _, id := range sortedMetricIDs(baseByID, candByID) {
		baseVal, baseOK := baseByID[id]
		candVal, candOK := candByID[id]
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n",
			id,
			formatMetricValue(baseVal, baseOK),
			formatMetricValue(candVal, candOK),
			formatPercentChange(baseVal, candVal, baseOK, candOK),
		)
	}

	fmt.Fprint(&b, "\nΔ% = (candidate − baseline) / baseline × 100. `n/a` when baseline is 0 or metric missing.\n")
	return b.String(), nil
}

func validateComparable(candidate, baseline *Scorecard) error {
	if baseline.Scenario.ID != candidate.Scenario.ID {
		return errors.Errorf("scenario id mismatch: baseline %q vs candidate %q",
			baseline.Scenario.ID, candidate.Scenario.ID)
	}
	if baseline.Scenario.Version != candidate.Scenario.Version {
		return errors.Errorf("scenario version mismatch: baseline %q vs candidate %q",
			baseline.Scenario.Version, candidate.Scenario.Version)
	}
	if baseline.Scenario.Maturity != candidate.Scenario.Maturity {
		return errors.Errorf("scenario maturity mismatch: baseline %q vs candidate %q",
			baseline.Scenario.Maturity, candidate.Scenario.Maturity)
	}
	return nil
}

func metricsByID(sc *Scorecard) map[string]float64 {
	out := make(map[string]float64, len(sc.Metrics))
	for _, m := range sc.Metrics {
		out[m.ID] = m.Value
	}
	return out
}

func sortedMetricIDs(maps ...map[string]float64) []string {
	ids := make(map[string]struct{})
	for _, m := range maps {
		for id := range m {
			ids[id] = struct{}{}
		}
	}
	sorted := make([]string, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	slices.Sort(sorted)
	return sorted
}

func formatMetricValue(value float64, ok bool) string {
	if !ok {
		return "—"
	}
	return fmt.Sprintf("%.4g", value)
}

func formatPercentChange(baseVal, candVal float64, baseOK, candOK bool) string {
	if !baseOK || !candOK {
		return "n/a"
	}
	if baseVal == 0 {
		if candVal == 0 {
			return "0.0%"
		}
		return "n/a"
	}
	return fmt.Sprintf("%+.1f%%", (candVal-baseVal)/baseVal*100)
}
