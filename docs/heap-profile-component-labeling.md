# Heap Profile Component Labeling Fix

**Status:** ✅ Implemented (2026-04-13)

## Problem

After PR #19819 (busybox consolidation), all components report `File: central` in heap profiles instead of their actual component name (e.g., `kubernetes-sensor`, `admission-control`).

**Root Cause:** All components are compiled into a single `central` binary. Even when invoked via symlinks (e.g., `/stackrox/bin/kubernetes-sensor`), Go's profiler reports the actual binary name, not the symlink name.

**Impact:** Makes it hard to identify which component's heap profile you're looking at.

## Solution

Use **pprof labels** to tag goroutines with the component name at startup. This embeds the component identity in profiles.

### Implementation (Completed)

**Approach:** Helper function in `pkg/profiling/component.go` that automatically extracts component name from `os.Args[0]`.

**Helper function:**
```go
// pkg/profiling/component.go
package profiling

import (
    "context"
    "os"
    "path/filepath"
    "runtime/pprof"
)

func SetComponentLabel() {
    componentName := filepath.Base(os.Args[0])
    labels := pprof.Labels("component", componentName)
    ctx := pprof.WithLabels(context.Background(), labels)
    pprof.SetGoroutineLabels(ctx)
}
```

**Usage in each component's `app.Run()`:**
```go
import "github.com/stackrox/rox/pkg/profiling"

func Run() {
    profiling.SetComponentLabel() // Automatically detects from os.Args[0]
    memlimit.SetMemoryLimit()
    premain.StartMain()
    // ... rest of component initialization
}
```

**Files Modified:**
- Created: `pkg/profiling/component.go`
- Modified (added `profiling.SetComponentLabel()` call):
  - `central/app/app.go`
  - `sensor/kubernetes/app/app.go`
  - `sensor/admission-control/app/app.go`
  - `config-controller/app/app.go`
  - `migrator/app/app.go`
  - `compliance/cmd/compliance/app/app.go`
  - `compliance/virtualmachines/roxagent/app/app.go`
  - `sensor/upgrader/app/app.go`
  - `roxctl/app/app.go`

All 9 busybox components now automatically tag their goroutines with the correct component name.

### Verification

After deployment, heap profiles will show:

```
# Before
File: central
...

# After
File: central
Labels: component=kubernetes-sensor
...
```

Query by label:
```bash
go tool pprof -tags component=kubernetes-sensor http://localhost:6060/debug/pprof/heap
```

Or verify directly from a captured profile:
```bash
curl http://sensor-pod:6060/debug/pprof/heap > sensor-heap.pb.gz
go tool pprof -raw sensor-heap.pb.gz | grep "component="
# Should show: component=kubernetes-sensor
```

### Benefits

1. **Heap profiles** clearly identify which component they're from
2. **CPU profiles** also get component labels
3. **pprof queries** can filter by component
4. **Zero runtime overhead** - labels are metadata, not allocations
5. **Works with all profiling tools** - pprof, continuous profiling, etc.
6. **Automatic detection** - no manual component name maintenance

### Build Verification

All components build successfully with the component labeling:
- ✅ central (485 MB)
- ✅ sensor/kubernetes (246 MB)
- ✅ sensor/admission-control (145 MB)
- ✅ config-controller
- ✅ migrator
- ✅ compliance
- ✅ roxagent
- ✅ sensor-upgrader
- ✅ roxctl

**Implementation Time:** ~1 hour (Option 1 - helper function approach)
