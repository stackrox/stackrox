package oomcheck

// Keeping files open requires half the time compared to os.ReadFile. I got 1974ns/op vs 5455ns/op in a simple
// read-and-parse-int benchmark. Pre-opening files allows to distinguish v1/v2 hierarchy once instead of on every call.
//
// I validated that file contents get refreshed as memory usage changes even though we keep the file open, i.e. there's
// no data correctness issue when doing it this way.
//
// Also, a small Python-based benchmark shows that just reading out cgroup (v1|v2) memory usage file takes
// 0.14ms+0.08ms user+system  time, therefore we shouldn't be too concerned with CPU load unless we start polling these
// files once per second or more often.

import (
	"io"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
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
	Open() error
	GetUsage() (MemoryUsage, error)
	Close()
}

type MemoryUsage struct {
	Used, Limit uint64
}

func NewMemoryUsageReader() MemoryUsageReader {
	return newWithRoot("/")
}

func newWithRoot(rootDir string) MemoryUsageReader {
	return &memoryUsageReaderImpl{
		v1StatFilePath:     path.Join(rootDir, cgroupV1StatFile),
		v1UsageFilePath:    path.Join(rootDir, cgroupV1UsageFile),
		procCgroupFilePath: path.Join(rootDir, procSelfCgroupFile),
		v2RootDirs: []string{
			path.Join(rootDir, cgroupV2Dir),
			path.Join(rootDir, cgroupV2FallbackDir),
		},
	}
}

type memoryUsageReaderImpl struct {
	v1StatFilePath     string
	v1UsageFilePath    string
	procCgroupFilePath string
	v2RootDirs         []string
	v1StatFile         *os.File
	v1UsageFile        *os.File
	v2CurrentFile      *os.File
	v2MaxFile          *os.File
}

func (r *memoryUsageReaderImpl) Open() error {
	var result *multierror.Error

	// First try cgroupv1 locations
	statFile, err := os.Open(r.v1StatFilePath)
	if err == nil {
		// memory.usage_in_bytes is optional, and we can get by without it in case of error.
		// TODO: test without it
		usageFile, _ := os.Open(r.v1UsageFilePath)
		r.v1StatFile = statFile
		r.v1UsageFile = usageFile
		return nil
	}

	result = multierror.Append(result,
		errors.Wrap(err, "cannot open cgroupv1 memory stat file, perhaps this system doesn't support cgroupv1"))

	// Try cgroupv2 in case v1 did not work
	subdir, err := getCgroupv2Subdir(r.procCgroupFilePath)
	if err != nil {
		result = multierror.Append(result, err)
		return result
	}

	var currentFile, maxFile *os.File
	// We probe hardcoded locations for cgroupv2 root mounts. These locations are based on what I've seen on real
	// systems and what seems to be mentioned on the internet as expected places. The problem is, though, that cgroupv2
	// can be mounted pretty much anywhere, not necessarily in the locations I've seen. To recognize that accurately, we
	// must parse /proc/self/mountinfo but that will complicate code even further, so I decided to do the simpler thing.
	for _, v2RootDir := range r.v2RootDirs {
		// TODO: test nil cases and combos
		if currentFile == nil {
			currentFile, err = os.Open(path.Join(v2RootDir, subdir, cgroupV2CurrentFile))
			if err != nil {
				result = multierror.Append(result, err)
			}
		}
		if maxFile == nil {
			maxFile, err = os.Open(path.Join(v2RootDir, subdir, cgroupV2MaxFile))
			if err != nil {
				result = multierror.Append(result, err)
			}
		}
	}

	if currentFile != nil && maxFile != nil {
		r.v2CurrentFile = currentFile
		r.v2MaxFile = maxFile
		return nil
	}

	if currentFile != nil {
		_ = currentFile.Close()
	}
	if maxFile != nil {
		_ = maxFile.Close()
	}

	// TODO: test how it comes out in the end
	return errors.Wrap(result, "neither cgroupv1 nor cgroupv2 memory information detected")
}

// Read out subdirectory in cgroup hierarchy for the specified process.
// See https://man7.org/linux/man-pages/man7/cgroups.7.html part for /proc/[pid]/cgroup describing expected file format.
func getCgroupv2Subdir(procFilePath string) (string, error) {
	data, err := os.ReadFile(procFilePath)
	if err != nil {
		return "", err
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
		return "", errors.Wrapf(errox.NotFound, "cgroup subdirectory record not found in the contents of %s", procFilePath)
	}
	return cgroupV2Subdir, nil
}

func (r *memoryUsageReaderImpl) Close() {
	// We ignore errors closing files because files were open only for reading.
	// There's no data corruption to worry about.
	_ = r.v1StatFile.Close()
	_ = r.v1UsageFile.Close()
	_ = r.v2CurrentFile.Close()
	_ = r.v2MaxFile.Close()
}

func (r *memoryUsageReaderImpl) GetUsage() (MemoryUsage, error) {
	if r.v1StatFile != nil {
		return r.getUsageCgroupV1()
	}
	if r.v2CurrentFile != nil && r.v2MaxFile != nil {
		return r.getUsageCgroupV2()
	}
	return MemoryUsage{}, errors.Wrap(errox.InvariantViolation, "cgroup memory usage information not available")
}

func (r *memoryUsageReaderImpl) getUsageCgroupV1() (MemoryUsage, error) {
	var buffer [4 * 1024]byte

	n, err := readFromStart(r.v1StatFile, buffer[:])
	if err != nil {
		return MemoryUsage{}, err
	}

	statStr := string(buffer[:n])
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
		// TODO: test absent value
		if parts[0] == "hierarchical_memory_limit" {
			result.Limit = val
		}
	}

	// In case memory.usage_in_bytes exists and shows value bigger than we computed above, take its value so that we're
	// more cautious when detecting pre-OOM condition.
	if r.v1UsageFile != nil {
		n, err = readFromStart(r.v1UsageFile, buffer[:])
		if err != nil {
			return MemoryUsage{}, err
		}

		val, err := strconv.ParseUint(strings.TrimSpace(string(buffer[:n])), 10, 64)
		if err != nil {
			return MemoryUsage{}, err
		}
		result.Used = mathutil.MaxUint64(result.Used, val)
	}

	return result, nil
}

func readFromStart(file *os.File, data []byte) (int, error) {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}
	n, err := file.Read(data)
	if err != nil {
		return 0, err
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (r *memoryUsageReaderImpl) getUsageCgroupV2() (MemoryUsage, error) {
	var buffer [64]byte

	var result MemoryUsage

	n, err := readFromStart(r.v2CurrentFile, buffer[:])
	if err != nil {
		return MemoryUsage{}, err
	}
	val, err := strconv.ParseUint(strings.TrimSpace(string(buffer[:n])), 10, 64)
	if err != nil {
		return MemoryUsage{}, err
	}
	result.Used = val

	n, err = readFromStart(r.v2MaxFile, buffer[:])
	if err != nil {
		return MemoryUsage{}, err
	}
	content := strings.TrimSpace(string(buffer[:n]))
	if content == maxUsage {
		result.Limit = maxValue
	} else {
		val, err = strconv.ParseUint(content, 10, 64)
		if err != nil {
			return MemoryUsage{}, err
		}
		result.Limit = val
	}

	return result, nil
}
