package oomcheck

import (
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/mathutil"
)

const (
	cgroupV1StatFile   = "/sys/fs/cgroup/memory/memory.stat"
	cgroupV1UsageFile  = "/sys/fs/cgroup/memory/memory.usage_in_bytes"
	procSelfCgroupFile = "/proc/self/cgroup"

	// TODO: document a better way
	cgroupV2Dir         = "/sys/fs/cgroup/unified"
	cgroupV2FallbackDir = "/sys/fs/cgroup"
	cgroupV2CurrentFile = "memory.current"
	cgroupV2MaxFile     = "memory.max"

	maxUsage = "max"
	maxValue = math.MaxUint64
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
	return newWithRoot("/")
}

func newWithRoot(rootDir string) MemoryUsageReader {
	return &memoryUsageReaderImpl{
		v1StatFile:     path.Join(rootDir, cgroupV1StatFile),
		v1UsageFile:    path.Join(rootDir, cgroupV1UsageFile),
		procCgroupFile: path.Join(rootDir, procSelfCgroupFile),
		v2RootDirs: []string{
			path.Join(rootDir, cgroupV2Dir),
			path.Join(rootDir, cgroupV2FallbackDir),
		},
	}
}

type memoryUsageReaderImpl struct {
	v1StatFile     string
	v1UsageFile    string
	procCgroupFile string
	v2RootDirs     []string
}

func (r *memoryUsageReaderImpl) GetUsage() (MemoryUsage, error) {
	result, err := r.getUsageCgroupV1()
	if err != nil && os.IsNotExist(err) {
		result, err = r.getUsageCgroupV2()
	}
	return result, err
}

func (r *memoryUsageReaderImpl) getUsageCgroupV1() (MemoryUsage, error) {
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

func (r *memoryUsageReaderImpl) getUsageCgroupV2() (MemoryUsage, error) {
	// TODO: cache things

	// See https://man7.org/linux/man-pages/man7/cgroups.7.html part for /proc/[pid]/cgroup
	data, err := os.ReadFile(r.procCgroupFile)
	if err != nil {
		return MemoryUsage{}, err
	}
	var cgroupV2Subdir string
	for _, line := range strings.Split(string(data), "\n") {
		subdir := strings.TrimPrefix(line, "0::")
		if subdir != line {
			cgroupV2Subdir = subdir
			break
		}
	}
	if cgroupV2Subdir == "" {
		return MemoryUsage{}, errors.Wrapf(errox.NotFound, "cgroup subdirectory record not found in the contents of %s", r.procCgroupFile)
	}

	var result MemoryUsage

	for _, v2Root := range r.v2RootDirs {
		dir := path.Join(v2Root, cgroupV2Subdir)

		data, err := os.ReadFile(path.Join(dir, cgroupV2CurrentFile))
		if err != nil {
			return MemoryUsage{}, err
		}

		val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		if err != nil {
			return MemoryUsage{}, err
		}
		result.Used = val

		data, err = os.ReadFile(path.Join(dir, cgroupV2MaxFile))
		if err != nil {
			return MemoryUsage{}, err
		}
		content := strings.TrimSpace(string(data))
		if content == maxUsage {
			result.Limit = maxValue
		} else {
			val, err = strconv.ParseUint(content, 10, 64)
			if err != nil {
				return MemoryUsage{}, err
			}
			result.Limit = val
		}
	}

	return result, nil
}
