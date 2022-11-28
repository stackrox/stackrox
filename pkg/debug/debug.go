package debug

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
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
	ticker    *time.Ticker
	lastDump  time.Time
}

func NewHeapProfiler(threshold float64, limit uint64) *HeapProfiler {
	return &HeapProfiler{
		Threshold: threshold,
		Limit:     limit,
		Backoff:   time.Second * 30,
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
		case <-runCheck:
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			if float64(mem.Alloc)/float64(p.Limit) > p.Threshold {
				if time.Since(p.lastDump) < p.Backoff {
					return
				}
				fmt.Println("implement heap dump here")
				p.lastDump = time.Now()
			}
		}
	}
}
