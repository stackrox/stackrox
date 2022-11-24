package oomcheck

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/mathutil"
)

const (
	cgroupV1Dir       = "/sys/fs/cgroup/memory"
	cgroupV1StatFile  = "memory.stat"
	cgroupV1UsageFile = "memory.usage_in_bytes"
)

var (
	// memoryUsageComponents are stats we use to compute used memory by cgroup (container).
	// See note in https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt , part 5.5 about computing exact memory
	// usage:
	// For efficiency, as other kernel components, memory cgroup uses some optimization
	// to avoid unnecessary cacheline false sharing. usage_in_bytes is affected by the
	// method and doesn't show 'exact' value of memory (and swap) usage, it's a fuzz
	// value for efficient access. (Of course, when necessary, it's synchronized.)
	// If you want to know more exact memory usage, you should use RSS+CACHE(+SWAP)
	// value in memory.stat(see 5.2).
	memoryUsageComponents = []string{"total_rss", "total_cache", "total_swap"}
)

type MemoryUsageReader interface {
	GetUsage() (MemoryUsage, error)
}

type MemoryUsage struct {
	Used, Limit uint64
}

func NewMemoryUsageReader() MemoryUsageReader {
	return newWithDirectory(cgroupV1Dir)
}

func newWithDirectory(v1dir string) MemoryUsageReader {
	return &memoryUsageReaderImpl{
		v1StatFile:  path.Join(v1dir, cgroupV1StatFile),
		v1UsageFile: path.Join(v1dir, cgroupV1UsageFile),
	}
}

type memoryUsageReaderImpl struct {
	v1StatFile  string
	v1UsageFile string
}

func (r *memoryUsageReaderImpl) GetUsage() (MemoryUsage, error) {
	// TODO: support cgroupV2
	data, err := os.ReadFile(r.v1StatFile)
	if err != nil {
		return MemoryUsage{}, err
	}
	statStr := string(data)
	var result MemoryUsage
	for _, line := range strings.Split(statStr, "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		val, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return MemoryUsage{}, err
		}

		for _, knownUsageComponent := range memoryUsageComponents {
			if parts[0] == knownUsageComponent {
				result.Used += val
				break
			}
		}
		if parts[0] == "hierarchical_memory_limit" {
			result.Limit = val
		}
	}

	data, err = os.ReadFile(r.v1UsageFile)
	if err != nil {
		return MemoryUsage{}, err
	}
	val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return MemoryUsage{}, err
	}
	result.Used = mathutil.MaxUint64(result.Used, val)

	return result, nil
}
