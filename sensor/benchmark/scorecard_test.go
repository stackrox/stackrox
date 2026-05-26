package benchmark

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScorecardJSONRoundTrip(t *testing.T) {
	original := Scorecard{
		Schema: ScorecardSchema,
		Scenario: ScenarioMeta{
			ID:       "steady-synthetic-dev-v0",
			Version:  "0",
			Maturity: "dev",
		},
		Run: RunMeta{
			GitSHA:      "abc123",
			GoVersion:   "go1.24.0",
			Runner:      "sensor-bench/v1",
			SyncWaitSec: 12.5,
			MeasureSec:  60,
			Success:     true,
		},
		Metrics: []Metric{
			{
				ID:        "k8s_events_ingress_per_sec",
				Phase:     "steady",
				Value:     123.45,
				Unit:      "1/s",
				Direction: "higher_is_better",
				Source: MetricSource{
					Prometheus:  "rox_sensor_k8s_events",
					Aggregation: "sum_delta_all_labels",
				},
			},
		},
		Extensions: map[string]any{
			"note": "dev run",
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Scorecard
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, original, decoded)
}
