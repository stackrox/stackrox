package profiling

import (
	"context"
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

// HeapProfiler is used to start a ticker that periodically checks if heap memory consumption
// exceed the ThresholdFraction, if so the heap gets dumped to a File in Directory.
type HeapProfiler struct {
	ThresholdFraction float64
	LimitBytes        uint64
	Backoff           time.Duration
	Directory         string
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

	return &HeapProfiler{
		ThresholdFraction: threshold,
		LimitBytes:        limit,
		Backoff:           time.Second * 30,
		Directory:         directory,
	}
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
				path := dumpFilePath(t, p.Directory)
				log.Debugf("heap memory usage exceeded threshold, dumping heap profile to: %v", path)
				if err := writeHeapProfile(t, path); err != nil {
					log.Debugf("error dumping heap: %s", err)
				}
				p.lastDump = time.Now()
			}
		}
	}
}

func writeHeapProfile(t time.Time, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "creating heap dump file at: %s", path)
	}

	return errors.Wrapf(pprof.Lookup("heap").WriteTo(file, 0), "writing heap profile to file at: %s", path)
}

func dumpFilePath(t time.Time, dir string) string {
	return path.Join(dir, t.Format("20060102T15-04-05"))
}
