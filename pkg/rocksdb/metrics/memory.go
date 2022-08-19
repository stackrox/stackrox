package metrics

import (
	"bytes"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/stringutils"
)

var log = logging.LoggerForModule()

func collectMemory() {
	file, err := os.Open("/proc/self/maps")
	if err != nil {
		panic(err)
	}
	parseMemory(file)
}

func parseMemory(file *os.File) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("could not read file %s: %v", file.Name(), err)
		return
	}

	resultsMap := make(map[string]int64)

	for _, line := range bytes.Split(data, []byte("\n")) {
		segments := bytes.Fields(line)
		if bytes.Equal(segments[3], []byte("00:00")) {
			continue
		}
		addrRange := segments[0]
		ranges := bytes.Split(addrRange, []byte("-"))

		start, err := strconv.ParseInt(string(ranges[0]), 16, 64)
		if err != nil {
			log.Errorf("could not convert %s: %v", string(ranges[0]), err)
			continue
		}
		end, err := strconv.ParseInt(string(ranges[1]), 16, 64)
		if err != nil {
			log.Errorf("could not convert %s: %v", string(ranges[1]), err)
			continue
		}
		file := string(segments[5])
		resultsMap[file] += end - start
	}
	groupByType := make(map[string]int64)
	for file, size := range resultsMap {
		var mmapType string
		if strings.Contains(file, ".so") {
			mmapType = "library"
		} else if strings.Contains(file, "central") {
			mmapType = "central"
		} else if strings.HasSuffix(file, "zap") || strings.HasSuffix(file, "root.bolt") {
			if strings.HasPrefix(file, "/tmp") {
				mmapType = "ephemeral-index"
			} else {
				if strings.HasPrefix(file, "/var/lib/stackrox/index") {
					file = strings.TrimPrefix(file, "/var/lib/stackrox/index/")
					mmapType = stringutils.GetUpTo(file, "/") + "-index"
				} else {
					mmapType = "dackbox-index"
				}
			}
		} else if strings.HasSuffix(file, "stackrox.db") {
			mmapType = "bolt"
		}
		if mmapType == "" {
			mmapType = "missing"
		}
		groupByType[mmapType] += size
	}
	for typ, size := range groupByType {
		mmapAllocations.With(prometheus.Labels{"Type": typ}).Set(float64(size))
	}
}
