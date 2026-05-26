package benchmark

const ScorecardSchema = "sensor-bench.scorecard/v1"

type Scorecard struct {
	Schema     string         `json:"schema"`
	Scenario   ScenarioMeta   `json:"scenario"`
	Run        RunMeta        `json:"run"`
	Metrics    []Metric       `json:"metrics"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

type ScenarioMeta struct {
	ID       string `json:"id"`
	Version  string `json:"version"`
	Maturity string `json:"maturity"` // "dev" | "benchmark"
}

type RunMeta struct {
	GitSHA      string  `json:"git_sha,omitempty"`
	GoVersion   string  `json:"go_version,omitempty"`
	Runner      string  `json:"runner,omitempty"`
	StartedAt   string  `json:"started_at,omitempty"`
	FinishedAt  string  `json:"finished_at,omitempty"`
	SyncWaitSec float64 `json:"sync_wait_sec,omitempty"`
	MeasureSec  float64 `json:"measure_sec,omitempty"`
	Success     bool    `json:"success"`
}

type Metric struct {
	ID        string       `json:"id"`
	Phase     string       `json:"phase"`
	Value     float64      `json:"value"`
	Unit      string       `json:"unit"`
	Direction string       `json:"direction"`
	Source    MetricSource `json:"source"`
}

type MetricSource struct {
	Prometheus  string `json:"prometheus"`
	Aggregation string `json:"aggregation"`
	Note        string `json:"note,omitempty"`
}
