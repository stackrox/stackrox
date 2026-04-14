package profiling

import (
	"context"
	"runtime/pprof"
)

// SetComponentLabel tags the current goroutine with the component name.
// This ensures heap and CPU profiles correctly identify which component
// they're from in busybox-style binaries.
//
// Call this early in app initialization, before starting goroutines.
func SetComponentLabel(componentName string) {
	labels := pprof.Labels("component", componentName)
	ctx := pprof.WithLabels(context.Background(), labels)
	pprof.SetGoroutineLabels(ctx)
}
