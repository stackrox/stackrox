package metrics

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/stackrox/pkg/stringutils"
)

const statFile = "/sys/fs/cgroup/cpu/cpu.stat"

// GatherThrottleMetricsForever reads the stat file and exposes the metrics for cpu throttling
func GatherThrottleMetricsForever(subsystem string) {
	go gatherThrottleMetricsForever(subsystem)
}

func gatherThrottleMetricsForever(subsystem string) {
	processCPUPeriods := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: PrometheusNamespace,
		Subsystem: subsystem,
		Name:      "process_cpu_nr_periods",
		Help:      "Number of CPU Periods (nr_periods)",
	})

	processCPUThrottledCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: PrometheusNamespace,
		Subsystem: subsystem,
		Name:      "process_cpu_nr_throttled",
		Help:      "Number of times the process was throttled",
	})

	processCPUThrottledTime := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: PrometheusNamespace,
		Subsystem: subsystem,
		Name:      "process_cpu_throttled_time",
		Help:      "Time in nanoseconds that the process has been throttled",
	})
	prometheus.MustRegister(
		processCPUPeriods,
		processCPUThrottledCount,
		processCPUThrottledTime,
	)

	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		data, err := os.ReadFile(statFile)
		if err != nil {
			log.Debugf("error reading file %s: %v", statFile, err)
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			metric, strValue := stringutils.Split2(line, " ")
			if strValue == "" {
				continue
			}
			value, err := strconv.ParseInt(strValue, 10, 64)
			if err != nil {
				log.Debugf("error parsing int64 in %s: %v", statFile, err)
				continue
			}

			switch metric {
			case "nr_periods":
				processCPUPeriods.Set(float64(value))
			case "nr_throttled":
				processCPUThrottledCount.Set(float64(value))
			case "throttled_time":
				processCPUThrottledTime.Set(float64(value))
			default:
				continue
			}
		}
	}
}
