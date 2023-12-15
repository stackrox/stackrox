package metrics

import (
	"bufio"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	cgroup2CPUStatFile = "/sys/fs/cgroup/cpu.stat"
	cgroupCPUStatFile  = "/sys/fs/cgroup/cpu/cpu.stat"
)

type cpuStats struct {
	periods       string
	throttled     string
	throttledTime string
	nanosFunc     func(float64) float64
}

var (
	cgroup2Stats = cpuStats{
		periods:       "nr_periods",
		throttled:     "nr_throttled",
		throttledTime: "throttled_usec", // microseconds
		nanosFunc:     func(f float64) float64 { return f * 1000 },
	}

	cgroupStats = cpuStats{
		periods:       "nr_periods",
		throttled:     "nr_throttled",
		throttledTime: "throttled_time", // nanoseconds
		nanosFunc:     func(f float64) float64 { return f },
	}
)

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

	statFile := cgroup2CPUStatFile
	stats := cgroup2Stats
	if _, err := os.Stat(statFile); errors.Is(err, fs.ErrNotExist) {
		statFile = cgroupCPUStatFile
		stats = cgroupStats
	}

	setProcessCPUThrottledTime := func(toNanos func(float64) float64) func(float64) {
		return func(f float64) {
			processCPUThrottledTime.Set(toNanos(f))
		}
	}

	log.Infof("gathering CPU throttle metrics from %s", statFile)
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		b, err := os.ReadFile(statFile)
		if err != nil {
			log.Debugf("error reading file %s: %v", statFile, err)
			continue
		}

		s := bufio.NewScanner(bytes.NewReader(b))
		for s.Scan() {
			metric, strValue, ok := strings.Cut(s.Text(), " ")
			if !ok {
				continue
			}

			var set func(float64)
			switch metric {
			case stats.periods:
				set = processCPUPeriods.Set
			case stats.throttled:
				set = processCPUThrottledCount.Set
			case stats.throttledTime:
				set = setProcessCPUThrottledTime(stats.nanosFunc)
			default:
			}
			if set == nil {
				continue
			}

			value, err := strconv.Atoi(strValue)
			if err != nil {
				log.Debugf("error parsing int64 in %s for metric %s: %v", statFile, metric, err)
				continue
			}

			set(float64(value))
		}
		if err := s.Err(); err != nil {
			log.Debugf("error scanning file %s: %v", statFile, err)
		}
	}
}
