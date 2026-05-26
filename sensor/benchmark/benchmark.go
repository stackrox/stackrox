package benchmark

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/tools/local-sensor/run"
)

const (
	runnerID          = "sensor-bench/v1"
	metricPhaseSteady = "steady"
	metricUnitPerSec  = "1/s"
	metricDirection   = "higher_is_better"
)

// Options holds optional overrides for RunScenario.
type Options struct{}

// RunScenario loads a scenario, runs an in-process Sensor, waits for initial sync,
// measures steady-state Prometheus counter rates, and returns a scorecard.
func RunScenario(ctx context.Context, scenarioDir string, _ Options) (*Scorecard, error) {
	scenario, err := LoadScenario(scenarioDir)
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()

	fakeCollector, err := workloadNeedsFakeCollector(scenario.ResolvedWorkloadPath())
	if err != nil {
		return nil, err
	}

	cfg := run.Config{
		FakeWorkloadFile:  scenario.ResolvedWorkloadPath(),
		PoliciesFile:      scenario.ResolvedPoliciesPath(),
		MetricsEnabled:    true,
		MetricsPort:       scenario.MetricsPort(),
		SkipCentralOutput: true,
		FakeCollector:     fakeCollector,
		NoCPUProfile:      true,
		NoMemProfile:      true,
		Verbose:           false,
	}

	handle, err := run.Run(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "starting sensor")
	}

	syncWaitStart := time.Now()
	syncCtx, syncCancel := context.WithTimeout(ctx, scenario.MaxSyncWait())
	syncErr := handle.WaitInitialSync(syncCtx)
	syncCancel()
	syncWaitSec := time.Since(syncWaitStart).Seconds()

	if syncErr != nil {
		stopWithTimeout(handle.Stop, 30*time.Second)
		sc := newScorecard(scenario, startedAt, time.Now(), syncWaitSec, 0, false)
		return sc, errors.Wrap(syncErr, "waiting for initial sync")
	}

	before, err := FetchMetrics(handle.MetricsURL)
	if err != nil {
		handle.Stop()
		return nil, errors.Wrap(err, "scraping metrics before steady window")
	}

	steadyDuration := scenario.SteadyDuration()
	if steadyDuration <= 0 {
		handle.Stop()
		return nil, errors.New("scenario steady phase duration must be positive")
	}

	select {
	case <-ctx.Done():
		handle.Stop()
		return nil, ctx.Err()
	case <-time.After(steadyDuration):
	}

	after, err := FetchMetrics(handle.MetricsURL)
	if err != nil {
		handle.Stop()
		return nil, errors.Wrap(err, "scraping metrics after steady window")
	}

	stopWithTimeout(handle.Stop, 2*time.Minute)
	finishedAt := time.Now()
	measureSec := steadyDuration.Seconds()

	return buildSteadyScorecard(scenario, before, after, startedAt, finishedAt, syncWaitSec, measureSec), nil
}

func stopWithTimeout(stop func(), timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

func buildSteadyScorecard(scenario *Scenario, before, after map[string]float64, startedAt, finishedAt time.Time, syncWaitSec, measureSec float64) *Scorecard {
	sc := newScorecard(scenario, startedAt, finishedAt, syncWaitSec, measureSec, true)
	sc.Metrics = steadyMetrics(before, after, measureSec)
	return sc
}

func newScorecard(scenario *Scenario, startedAt, finishedAt time.Time, syncWaitSec, measureSec float64, success bool) *Scorecard {
	return &Scorecard{
		Schema: ScorecardSchema,
		Scenario: ScenarioMeta{
			ID:       scenario.Metadata.Name,
			Version:  scenario.Metadata.Version,
			Maturity: scenario.Maturity(),
		},
		Run: RunMeta{
			GitSHA:      gitSHA(),
			GoVersion:   runtime.Version(),
			Runner:      runnerID,
			StartedAt:   startedAt.UTC().Format(time.RFC3339),
			FinishedAt:  finishedAt.UTC().Format(time.RFC3339),
			SyncWaitSec: syncWaitSec,
			MeasureSec:  measureSec,
			Success:     success,
		},
	}
}

func steadyMetrics(before, after map[string]float64, measureSec float64) []Metric {
	return []Metric{
		{
			ID:        "k8s_events_ingress_per_sec",
			Phase:     metricPhaseSteady,
			Value:     RatePerSec(SumCounterDelta("rox_sensor_k8s_events", before, after), measureSec),
			Unit:      metricUnitPerSec,
			Direction: metricDirection,
			Source: MetricSource{
				Prometheus:  "rox_sensor_k8s_events",
				Aggregation: "sum_delta_all_labels",
			},
		},
		{
			ID:    "k8s_sensor_events_egress_per_sec",
			Phase: metricPhaseSteady,
			Value: RatePerSec(
				SumCounterDeltaFilteredResourceIn(
					"rox_sensor_sensor_events",
					before,
					after,
					map[string]string{"type": "total"},
					k8sSensorEventEgressResources,
				),
				measureSec,
			),
			Unit:      metricUnitPerSec,
			Direction: metricDirection,
			Source: MetricSource{
				Prometheus:  "rox_sensor_sensor_events",
				Aggregation: "sum_delta_filtered_resource_in",
				Note:        "type=total, resource in Deployment,Pod,Namespace,Node,ServiceAccount,Role,RoleBinding,NetworkPolicy,Secret,Image",
			},
		},
		{
			ID:        "collector_msgs_ingress_per_sec",
			Phase:     metricPhaseSteady,
			Value:     RatePerSec(SumCounterDelta("rox_sensor_host_connections_msgs_received_per_node_total", before, after), measureSec),
			Unit:      metricUnitPerSec,
			Direction: metricDirection,
			Source: MetricSource{
				Prometheus:  "rox_sensor_host_connections_msgs_received_per_node_total",
				Aggregation: "sum_delta_all_hostname",
			},
		},
		{
			ID:        "network_flow_updates_egress_per_sec",
			Phase:     metricPhaseSteady,
			Value:     RatePerSec(SumCounterDelta("rox_sensor_network_flow_manager_num_sent_to_central_total", before, after), measureSec),
			Unit:      metricUnitPerSec,
			Direction: metricDirection,
			Source: MetricSource{
				Prometheus:  "rox_sensor_network_flow_manager_num_sent_to_central_total",
				Aggregation: "sum_delta_all_object",
			},
		},
		{
			ID:        "process_signals_ingress_per_sec",
			Phase:     metricPhaseSteady,
			Value:     RatePerSec(SumCounterDelta("rox_sensor_process_signals_received_total", before, after), measureSec),
			Unit:      metricUnitPerSec,
			Direction: metricDirection,
			Source: MetricSource{
				Prometheus:  "rox_sensor_process_signals_received_total",
				Aggregation: "delta",
			},
		},
		{
			ID:    "process_indicator_events_egress_per_sec",
			Phase: metricPhaseSteady,
			Value: RatePerSec(
				SumCounterDeltaFiltered(
					"rox_sensor_sensor_events",
					before,
					after,
					map[string]string{"type": "total", "resource": "ProcessIndicator"},
				),
				measureSec,
			),
			Unit:      metricUnitPerSec,
			Direction: metricDirection,
			Source: MetricSource{
				Prometheus:  "rox_sensor_sensor_events",
				Aggregation: "sum_delta_filtered",
				Note:        "type=total resource=ProcessIndicator",
			},
		},
	}
}

// FetchMetrics performs an HTTP GET on url and parses Prometheus text exposition.
func FetchMetrics(url string) (map[string]float64, error) {
	if url == "" {
		return nil, errors.New("metrics URL is empty")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating metrics request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "fetching metrics")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics fetch: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading metrics body")
	}
	return ParseMetricFamilies(body)
}

func gitSHA() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key != "vcs.revision" {
			continue
		}
		rev := setting.Value
		if len(rev) > 12 {
			return rev[:12]
		}
		return rev
	}
	return ""
}
