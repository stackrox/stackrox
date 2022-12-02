package profiling

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func TestHeapDump(t *testing.T) {
	tmpDir := t.TempDir()

	var limitBytes int64 = 2 // to be sure the test blows this limit
	p := NewHeapProfiler(0.80, uint64(limitBytes), tmpDir)
	runCheck := make(chan time.Time)
	ctx, cancelCtx := context.WithCancel(context.Background())
	now := time.Now()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		p.dumpHeapOnThreshhold(ctx, runCheck)
		wg.Done()
	}()

	runCheck <- now
	cancelCtx()
	wg.Wait()

	expectedFilePath := dumpFilePath(now, tmpDir)
	if _, err := os.Stat(expectedFilePath); err != nil {
		t.Fatalf("expected heap dump file: %s not found", expectedFilePath)
	}
}
