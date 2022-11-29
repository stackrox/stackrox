package debug

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func TestHeapDump(t *testing.T) {
	tmpDir := t.TempDir()

	p := NewHeapProfiler(0.80, 2, tmpDir)
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
		t.Fatalf("expected file: %s not found", expectedFilePath)
	}
}
