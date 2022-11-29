package debug

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"time"
)

// FreeOSMemory runs a GC and then tries to relinquish as much memory back to the OS as possible
func FreeOSMemory() {
	debug.FreeOSMemory()
}

type HeapProfiler struct {
	Threshold float64
	Limit     uint64
	Backoff   time.Duration
	Directory string
	ticker    *time.Ticker
	lastDump  time.Time
}

func NewHeapProfiler(threshold float64, limit uint64, directory string) *HeapProfiler {
	return &HeapProfiler{
		Threshold: threshold,
		Limit:     limit,
		Backoff:   time.Second * 30,
		Directory: directory,
	}
}

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
			if float64(mem.Alloc)/float64(p.Limit) > p.Threshold {
				if time.Since(p.lastDump) < p.Backoff {
					return
				}
				if err := writeHeapProfile(t, p.Directory); err != nil {
					fmt.Printf("TODO: log errors correctly: %v", err)
				}
				p.lastDump = time.Now()
			}
		}
	}
}

func writeHeapProfile(t time.Time, dir string) error {
	path := dumpFilePath(t, dir)
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	return pprof.Lookup("heap").WriteTo(file, 0)
}

func dumpFilePath(t time.Time, dir string) string {
	return fmt.Sprintf("%s/%s.dump", dir, t.Format("20060102T15-04-05"))
}
