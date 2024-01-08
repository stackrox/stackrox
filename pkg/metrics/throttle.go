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
	cgroup2CPUStatFile = "sys/fs/cgroup/cpu.stat"
	cgroupCPUStatFile  = "sys/fs/cgroup/cpu/cpu.stat"
)

type cpuStats struct {
	periods       string
	throttled     string
	throttledTime string
	nanosFunc     func(float64) float64
}

type cpuStatValues struct {
	periods       int
	throttled     int
	throttledTime int
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

	fsys := os.DirFS("/")
	statFile := cgroup2CPUStatFile
	stats := cgroup2Stats
	if _, err := fs.Stat(fsys, statFile); errors.Is(err, fs.ErrNotExist) {
		statFile = cgroupCPUStatFile
		stats = cgroupStats
	}

	log.Infof("gathering CPU throttle metrics from %s", statFile)
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		b, err := fs.ReadFile(fsys, statFile)
		if err != nil {
			log.Debugf("error reading file %s: %v", statFile, err)
			continue
		}

		statVals := parseStats(statFile, b, stats)
		if statVals.periods != -1 {
			processCPUPeriods.Set(float64(statVals.periods))
		}
		if statVals.throttled != -1 {
			processCPUThrottledCount.Set(float64(statVals.throttled))
		}
		if statVals.throttledTime != -1 {
			processCPUThrottledTime.Set(stats.nanosFunc(float64(statVals.throttledTime)))
		}
	}
}

// parseStats parses the desired cpuStats from the relevant stats files represented by b.
//
// If a stat has value -1, it was unable to be read and should be ignored.
func parseStats(path string, b []byte, stats cpuStats) cpuStatValues {
	statVals := cpuStatValues{
		periods:       -1,
		throttled:     -1,
		throttledTime: -1,
	}

	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		metric, strValue, ok := strings.Cut(s.Text(), " ")
		if !ok {
			continue
		}

		switch metric {
		case stats.periods:
		case stats.throttled:
		case stats.throttledTime:
		default:
			continue
		}

		value, err := strconv.Atoi(strValue)
		if err != nil {
			log.Debugf("error parsing int64 in %s for metric %s: %v", path, metric, err)
			continue
		}

		switch metric {
		case stats.periods:
			statVals.periods = value
		case stats.throttled:
			statVals.throttled = value
		case stats.throttledTime:
			statVals.throttledTime = value
		default:
		}
	}
	if err := s.Err(); err != nil {
		log.Debugf("error scanning file %s: %v", path, err)
	}

	return statVals
}
