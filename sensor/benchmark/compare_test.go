package benchmark

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareScorecards(t *testing.T) {
	t.Parallel()

	base := sampleScorecard("steady-synthetic-dev-v0", "0", "dev", 10.0, 100.0)
	cand := sampleScorecard("steady-synthetic-dev-v0", "0", "dev", 12.0, 80.0)

	md, err := CompareScorecards(cand, base)
	require.NoError(t, err)
	require.Contains(t, md, "**Ingress**")
	require.Contains(t, md, "**Egress**")
	require.Contains(t, md, "`k8s_events_ingress_per_sec`")
	require.Contains(t, md, "`process_signals_ingress_per_sec`")
	require.Contains(t, md, "+20.0%")
	require.Contains(t, md, "-20.0%")
}

func TestCompareScorecards_mismatchScenario(t *testing.T) {
	t.Parallel()

	base := sampleScorecard("steady-synthetic-dev-v0", "0", "dev", 1, 1)
	cand := sampleScorecard("other-scenario", "0", "dev", 1, 1)

	_, err := CompareScorecards(cand, base)
	require.Error(t, err)
	require.Contains(t, err.Error(), "scenario id mismatch")
}

func TestCompareScorecards_zeroBaseline(t *testing.T) {
	t.Parallel()

	base := sampleScorecard("steady-synthetic-dev-v0", "0", "dev", 0, 5)
	cand := sampleScorecard("steady-synthetic-dev-v0", "0", "dev", 1, 5)

	md, err := CompareScorecards(cand, base)
	require.NoError(t, err)
	require.Contains(t, md, "n/a")
}

func TestLoadScorecard_roundTrip(t *testing.T) {
	t.Parallel()

	original := sampleScorecard("steady-synthetic-dev-v0", "0", "dev", 5.3, 100)
	path := filepath.Join(t.TempDir(), "scorecard.json")
	data, err := json.Marshal(original)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	loaded, err := LoadScorecard(path)
	require.NoError(t, err)
	require.Equal(t, original.Scenario, loaded.Scenario)
	require.Len(t, loaded.Metrics, len(original.Metrics))
}

func TestLoadScorecard_invalidSchema(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "scorecard.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"schema":"other/v0","scenario":{"id":"x","version":"0","maturity":"dev"},"run":{"success":true},"metrics":[]}`), 0o644))
	_, err := LoadScorecard(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported schema")
}

func sampleScorecard(id, version, maturity string, k8sRate, processRate float64) *Scorecard {
	return &Scorecard{
		Schema: ScorecardSchema,
		Scenario: ScenarioMeta{
			ID:       id,
			Version:  version,
			Maturity: maturity,
		},
		Run: RunMeta{
			Success:    true,
			MeasureSec: 60,
		},
		Metrics: []Metric{
			{ID: "k8s_events_ingress_per_sec", Value: k8sRate},
			{ID: "process_signals_ingress_per_sec", Value: processRate},
		},
	}
}
