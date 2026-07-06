package main

import (
	"fmt"
	"math"
	"strings"
	"text/tabwriter"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// Metric names exposed by sensor/common/virtualmachine/metrics, duplicated
// here as string constants since the summary reads them back out of the
// Prometheus registry rather than from the metric objects directly (the
// registry is what a real deployment would be scraped through, so this
// keeps the summary honest about what's actually being exported).
const (
	metricCycleDuration = "rox_sensor_vsock_pull_cycle_duration_seconds"
	metricCyclesTotal   = "rox_sensor_vsock_pull_cycles_total"
	metricVMsInCycle    = "rox_sensor_vsock_pull_vms_in_cycle"
	metricRequestsTotal = "rox_sensor_vsock_pull_requests_total"
	metricDialDuration  = "rox_sensor_vsock_pull_dial_duration_seconds"
	metricReadDuration  = "rox_sensor_vsock_pull_read_duration_seconds"
	metricTotalDuration = "rox_sensor_vsock_pull_total_duration_seconds"
)

// printSummary gathers the vsock_pull_* metrics from the default Prometheus
// registry and prints a compact, human-readable report. Percentiles are
// linearly interpolated from histogram bucket boundaries -- an approximation
// bounded by the bucket width, not an exact percentile over raw samples. When
// a histogram's first bucket already contains (almost) all samples, there is
// no resolution left to interpolate a meaningful percentile at all; see
// printLatencyRow / unresolvedBucketThreshold.
//
// cfg is echoed verbatim at the top of the report so that every result is
// self-describing and reproducible -- no guessing what flags produced a given
// number when the summary is copy-pasted elsewhere later.
func printSummary(cfg runConfig) {
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		log.Errorf("vmscraper-loadtest: gathering metrics for summary: %v", err)
		return
	}
	byName := make(map[string]*dto.MetricFamily, len(families))
	for _, f := range families {
		byName[f.GetName()] = f
	}

	var b strings.Builder
	b.WriteString("\n=== vmscraper-loadtest summary ===\n")
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)

	fmt.Fprintf(tw, "Config:\t%s\n", formatConfig(cfg))

	cycles := gaugeOrCounterValue(byName[metricCyclesTotal])
	vmsInCycle := gaugeOrCounterValue(byName[metricVMsInCycle])
	if hist := soleHistogram(byName[metricCycleDuration]); hist != nil && hist.GetSampleCount() > 0 {
		avg := hist.GetSampleSum() / float64(hist.GetSampleCount())
		fmt.Fprintf(tw, "Cycles completed:\t%d\n", uint64(cycles))
		fmt.Fprintf(tw, "VMs per cycle:\t%d\n", uint64(vmsInCycle))
		fmt.Fprintf(tw, "Cycle duration:\tavg=%s  (sum=%s over %d cycles)\n",
			formatDuration(avg), formatDuration(hist.GetSampleSum()), hist.GetSampleCount())
	} else {
		fmt.Fprintf(tw, "Cycles completed:\t%d (no completed cycles yet)\n", uint64(cycles))
	}

	if fam := byName[metricRequestsTotal]; fam != nil {
		fmt.Fprintf(tw, "Requests by status:\t%s\n", formatLabeledCounters(fam, "status"))
	}

	printLatencyRow(tw, byName[metricDialDuration], "Dial latency")
	printLatencyRow(tw, byName[metricReadDuration], "Read latency")
	printLatencyRow(tw, byName[metricTotalDuration], "Total latency (dial+read+send)")

	_ = tw.Flush()
	fmt.Println(b.String())
}

// unresolvedBucketThreshold: if this fraction (or more) of samples land in
// the histogram's first bucket, there isn't enough resolution left to tell
// p50 apart from p99 -- interpolating a percentile in that regime produces a
// number that is pure arithmetic on the bucket width, not a measurement (see
// docs/superpowers/reports/2026-07-03-vsock-pull-loadtest-day1-report.md,
// "Issue found" section). Report the honest bound instead of a fake number.
const unresolvedBucketThreshold = 0.99

func printLatencyRow(tw *tabwriter.Writer, fam *dto.MetricFamily, label string) {
	hist := soleHistogram(fam)
	if hist == nil || hist.GetSampleCount() == 0 {
		fmt.Fprintf(tw, "%s:\t(no samples)\n", label)
		return
	}

	if fraction, upperBound := firstBucketStats(hist); fraction >= unresolvedBucketThreshold {
		fmt.Fprintf(tw, "%s:\t<%s for %.0f%% of samples (below histogram resolution, n=%d)\n",
			label, formatDuration(upperBound), fraction*100, hist.GetSampleCount())
		return
	}

	fmt.Fprintf(tw, "%s:\tp50=%s  p90=%s  p99=%s  (n=%d)\n",
		label,
		formatDuration(histogramPercentile(hist, 0.50)),
		formatDuration(histogramPercentile(hist, 0.90)),
		formatDuration(histogramPercentile(hist, 0.99)),
		hist.GetSampleCount())
}

// firstBucketStats returns the fraction of samples that landed in the
// histogram's first (smallest-upper-bound) bucket, and that bucket's upper
// bound. Returns (0, 0) if there are no buckets or no samples.
func firstBucketStats(h *dto.Histogram) (fraction, upperBound float64) {
	buckets := h.GetBucket()
	total := h.GetSampleCount()
	if len(buckets) == 0 || total == 0 {
		return 0, 0
	}
	return float64(buckets[0].GetCumulativeCount()) / float64(total), buckets[0].GetUpperBound()
}

func soleHistogram(fam *dto.MetricFamily) *dto.Histogram {
	if fam == nil || len(fam.GetMetric()) == 0 {
		return nil
	}
	return fam.GetMetric()[0].GetHistogram()
}

func gaugeOrCounterValue(fam *dto.MetricFamily) float64 {
	if fam == nil || len(fam.GetMetric()) == 0 {
		return 0
	}
	m := fam.GetMetric()[0]
	if m.GetGauge() != nil {
		return m.GetGauge().GetValue()
	}
	return m.GetCounter().GetValue()
}

// formatConfig renders every flag that affects a run's results, using the
// same names as the CLI flags, so a printed summary can be turned back into
// the exact command line that produced it.
func formatConfig(cfg runConfig) string {
	return fmt.Sprintf(
		"num-vms=%d num-packages=%d poll-interval=%s concurrency=%d per-vm-timeout=%s dial-latency=%s rescan-interval=%s always-changed=%t duration=%s",
		cfg.numVMs, cfg.numPackages, cfg.pollInterval, cfg.concurrency, cfg.perVMTimeout,
		cfg.dialLatency, cfg.rescanInterval, cfg.alwaysChanged, cfg.duration)
}

// formatLabeledCounters renders every series of a labeled counter vec as
// "labelValue=count" pairs, e.g. "success=9998 dial_error=2".
func formatLabeledCounters(fam *dto.MetricFamily, labelName string) string {
	parts := make([]string, 0, len(fam.GetMetric()))
	for _, m := range fam.GetMetric() {
		value := "?"
		for _, lp := range m.GetLabel() {
			if lp.GetName() == labelName {
				value = lp.GetValue()
			}
		}
		parts = append(parts, fmt.Sprintf("%s=%d", value, uint64(m.GetCounter().GetValue())))
	}
	return strings.Join(parts, "  ")
}

// histogramPercentile linearly interpolates the p-th percentile (0..1) from
// a Prometheus histogram's cumulative bucket counts.
func histogramPercentile(h *dto.Histogram, p float64) float64 {
	total := h.GetSampleCount()
	if total == 0 {
		return 0
	}
	target := p * float64(total)

	var prevUpper float64
	var prevCount uint64
	for _, bucket := range h.GetBucket() {
		count := bucket.GetCumulativeCount()
		upper := bucket.GetUpperBound()
		if float64(count) >= target {
			if math.IsInf(upper, 1) || count == prevCount {
				return prevUpper
			}
			frac := (target - float64(prevCount)) / float64(count-prevCount)
			return prevUpper + frac*(upper-prevUpper)
		}
		prevUpper, prevCount = upper, count
	}
	return prevUpper
}

func formatDuration(seconds float64) string {
	switch {
	case seconds < 1:
		return fmt.Sprintf("%.1fms", seconds*1000)
	default:
		return fmt.Sprintf("%.2fs", seconds)
	}
}
