package oomcheck

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	checkInterval  = 3 * time.Second
	alarmThreshold = 11
)

func StartPreOomCheck() *concurrency.Signal {
	stopSignal := concurrency.NewSignal()
	go func() {
		ticker := time.NewTicker(checkInterval)
		for {
			select {
			case <-stopSignal.Done():
				return
			case <-ticker.C:
			}

			checkUsageAndReport()
		}
	}()
	return &stopSignal
}

func checkUsageAndReport() {
	stat, err := NewMemoryUsageReader().GetUsage()
	if err != nil {
		// TODO: smarter logging
		return
	}
	usedPercent := stat.Used * 100 / stat.Limit
	if usedPercent > alarmThreshold {
		// TODO: consider scientific notation
		log.Warnf("Memory usage %d%% exceeds %d%% threshold. The container is at risk of being OOM killed. Used memory %d bytes, limit %d bytes.",
			usedPercent, alarmThreshold, stat.Used, stat.Limit,
		)
	}
}

//limit, err := readScalar("/sys/fs/cgroup/memory/memory.limit_in_bytes")
//func readScalar(path string) (uint64, error) {
//	data, err := os.ReadFile(path)
//	if err != nil {
//		return 0, err
//	}
//	limitStr := string(data)
//	limit, err := strconv.ParseUint(limitStr, 10, 64)
//	if err != nil {
//		return 0, err
//	}
//	return limit, nil
//}
