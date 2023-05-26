package metrics

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/quay/zlog"
)

const (
	statFile = "/sys/fs/cgroup/cpu/cpu.stat"

	tickerPeriod = 30 * time.Second
)

var (
	processCPUPeriods = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_cpu_nr_periods",
		Help: "Number of CPU Periods (nr_periods)",
	})
	processCPUThrottledCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_cpu_nr_throttled",
		Help: "Number of times the process was throttled",
	})
	processCPUThrottledTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_cpu_throttled_time",
		Help: "Time in nanoseconds that the process has been throttled",
	})
)

func init() {
	prometheus.MustRegister(
		processCPUPeriods,
		processCPUThrottledCount,
		processCPUThrottledTime,
	)
}

// gatherThrottleMetrics gathers prometheus throttle metrics for
// the duration of the given context.
func gatherThrottleMetrics(ctx context.Context) {
	// Start reading the stats file periodically.
	ticker := time.NewTicker(tickerPeriod)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
		}

		f, err := os.Open(statFile)
		if err != nil {
			zlog.Warn(ctx).
				Err(err).
				Msg("Could not open cgroup CPU stats file")

			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			metric, value, found := strings.Cut(line, " ")
			if !found {
				zlog.Debug(ctx).
					Str("Stat line", line).
					Msg("Unexpected format for cgroup CPU stats line")

				continue
			}

			v, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				zlog.Debug(ctx).
					Str("Stat line", line).
					Msg("Unexpected format for cgroup CPU stats value")

				continue
			}

			switch metric {
			case "nr_periods":
				processCPUPeriods.Set(float64(v))
			case "nr_throttled":
				processCPUThrottledCount.Set(float64(v))
			case "throttled_time":
				processCPUThrottledTime.Set(float64(v))
			default:
				zlog.Debug(ctx).
					Str("Stat line", line).
					Msg("Unexpected cgroup CPU stat")
			}
		}

		if err := scanner.Err(); err != nil {
			zlog.Warn(ctx).
				Err(err).
				Msg("Could not read cgroup CPU stats file")

			continue
		}
	}
}
