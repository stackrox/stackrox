package profiling

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	fifoDefaultMaxFileCount = 10
	heapdumpSubfolderName   = "heapdump"
)

type fifoDir struct {
	maxFileCount int
	dirPath      string
}

func (fd fifoDir) Create(fileName string) (*os.File, error) {
	err := os.MkdirAll(fd.dirPath, os.ModePerm)
	if err != nil {
		if !os.IsExist(err) {
			return nil, errors.Wrapf(err, "creating directory: %s", fd.dirPath)
		}
	}

	entries, err := ioutil.ReadDir(fd.dirPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading directory: %s", fd.dirPath)
	}

	if len(entries) >= fd.maxFileCount {
		var oldestEntryIndex int

		for i, e := range entries {
			oldestEntryInfo := entries[oldestEntryIndex]

			if e.ModTime().Before(oldestEntryInfo.ModTime()) {
				oldestEntryIndex = i
			}
		}

		rmPath := path.Join(fd.dirPath, entries[oldestEntryIndex].Name())
		os.Remove(rmPath)
	}

	filePath := path.Join(fd.dirPath, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "creating file: %s", filePath)
	}

	return file, nil
}

// HeapProfiler is used to start a ticker that periodically checks if heap memory consumption
// exceed the ThresholdFraction, if so the heap gets dumped to a File in Directory.
type HeapProfiler struct {
	ThresholdFraction float64
	LimitBytes        uint64
	Backoff           time.Duration
	directory         *fifoDir
	ticker            *time.Ticker
	lastDump          time.Time
}

// NewHeapProfiler creates a new instance of HeapProfiler setting the given values.
// If 0 is provides as limit the limit is set to MaxUint64, thus the heap dump will never run.
func NewHeapProfiler(threshold float64, limit uint64, directory string) *HeapProfiler {
	// default to MaxUint64 to prevent division through 0
	if limit == 0 {
		limit = math.MaxUint64
	}

	fd := &fifoDir{
		dirPath:      path.Join(directory, heapdumpSubfolderName),
		maxFileCount: fifoDefaultMaxFileCount,
	}

	return &HeapProfiler{
		ThresholdFraction: threshold,
		LimitBytes:        limit,
		Backoff:           time.Second * 30,
		directory:         fd,
	}
}

// SetDirectory sets the target directory for dumps written by HeapProfiler
func (p *HeapProfiler) SetDirectory(dir string) {
	if p.directory == nil {
		p.directory = &fifoDir{
			maxFileCount: fifoDefaultMaxFileCount,
		}
	}

	p.directory.dirPath = path.Join(dir, heapdumpSubfolderName)
}

// DumpHeapOnThreshhold starts a time.Ticker to check heap usage with the given interval
func (p *HeapProfiler) DumpHeapOnThreshhold(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	p.dumpHeapOnThreshhold(ctx, ticker.C)
}

func (p *HeapProfiler) dumpHeapOnThreshhold(ctx context.Context, runCheck <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-runCheck:
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			if float64(mem.Alloc)/float64(p.LimitBytes) > p.ThresholdFraction {
				if time.Since(p.lastDump) < p.Backoff {
					return
				}
				fileName := fmt.Sprintf("%s.dump", t.Format("20060102T15-04-05"))
				log.Debugf("heap memory usage exceeded threshold, dumping heap profile to: %v", fileName)
				if err := p.writeHeapProfile(fileName); err != nil {
					log.Errorf("error dumping heap on memory threshold: %s", err)
				}
				p.lastDump = time.Now()
			}
		}
	}
}

func (p *HeapProfiler) writeHeapProfile(fileName string) error {
	file, err := p.directory.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "creating heap dump file at: %s", fileName)
	}

	return errors.Wrapf(pprof.Lookup("heap").WriteTo(file, 0), "writing heap profile to file at: %s", fileName)
}
