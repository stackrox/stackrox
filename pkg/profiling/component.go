package profiling

import (
	"context"
	"os"
	"path/filepath"
	"runtime/pprof"
)

// SetComponentLabel tags the current goroutine with the component name
// derived from os.Args[0]. This ensures heap and CPU profiles correctly
// identify which component they're from in busybox-style binaries.
//
// Call this early in app initialization, before starting goroutines.
func SetComponentLabel() {
	componentName := filepath.Base(os.Args[0])
	labels := pprof.Labels("component", componentName)
	ctx := pprof.WithLabels(context.Background(), labels)
	pprof.SetGoroutineLabels(ctx)
}
