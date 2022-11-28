package debug

import (
	"context"
	"testing"
)

func TestHeapDump(t *testing.T) {
	p := NewProfiler()
	p.DumpHeapOnThreshhold(context.Background(), 0.80)
}
