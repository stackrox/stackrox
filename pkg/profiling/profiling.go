package profiling

import (
	"context"
	"fmt"
	"path"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log                   = logging.LoggerForModule()
	heapdumpSubfolderName = "heapdump"
	// DefaultHeapProfilerBackoff is the default setting for the time to wait between heap dumps when hitting threshold in seconds
	DefaultHeapProfilerBackoff = time.Second * 30
)

// HeapProfiler is used to start a ticker that periodically checks if heap memory consumption
// exceed the thresholdFraction, if so the heap gets dumped to a file in Directory subject to Backoff.
type HeapProfiler struct {
	backoff           time.Duration
	thresholdFraction float64
	limitBytes        uint64
	directory         *fifoDir
	lastDump          time.Time
}

// NewHeapProfiler creates a new instance of HeapProfiler setting the given values.
// It appends a subdirectory to the directory to prevent acidental deletion of user files.
// If 0 is provides as limitBytes the limitBytes is set to 1, thus the heap dump will always run.
// backoff limits the maximum frequency of creating heap dumps
func NewHeapProfiler(thresholdFraction float64, limitBytes uint64, directory string, backoff time.Duration) *HeapProfiler {
	// default to 1 to prevent division through 0
	if limitBytes == 0 {
		limitBytes = 1
	}

	fd := &fifoDir{
		dirPath:      path.Join(directory, heapdumpSubfolderName),
		maxFileCount: fifoDefaultMaxFileCount,
	}

	return &HeapProfiler{
		thresholdFraction: thresholdFraction,
		limitBytes:        limitBytes,
		backoff:           backoff,
		directory:         fd,
	}
}

// DumpHeapOnThreshhold runs for as long as ctx is valid, checking heap usage with the given interval.
// Maximum frequency of creating heap dumps is limited by the configured backoff.
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
			if float64(mem.Alloc)/float64(p.limitBytes) > p.thresholdFraction {
				if time.Since(p.lastDump) < p.backoff {
					continue // this will skip all code below and jump to start of the for loop
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
