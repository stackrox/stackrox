# Heap Profile Component Labeling Fix

## Problem

After PR #19819 (busybox consolidation), all components report `File: central` in heap profiles instead of their actual component name (e.g., `kubernetes-sensor`, `admission-control`).

**Root Cause:** All components are compiled into a single `central` binary. Even when invoked via symlinks (e.g., `/stackrox/bin/kubernetes-sensor`), Go's profiler reports the actual binary name, not the symlink name.

**Impact:** Makes it hard to identify which component's heap profile you're looking at.

## Solution

Use **pprof labels** to tag goroutines with the component name at startup. This embeds the component identity in profiles.

### Implementation

Add component labeling in each `component/app/app.go` Run() function:

```go
import (
    "context"
    "runtime/pprof"
)

func Run() {
    // Set component label for profiling
    pprof.Do(context.Background(), pprof.Labels("component", "kubernetes-sensor"), func(ctx context.Context) {
        // All existing initialization and runtime code runs in this labeled context
        memlimit.SetMemoryLimit()
        premain.StartMain()

        initMetrics()

        // ... rest of Run() logic
    })
}
```

Or use `pprof.SetGoroutineLabels()` for global tagging:

```go
func Run() {
    // Tag this goroutine (and all children) with component name
    labels := pprof.Labels("component", "kubernetes-sensor")
    pprof.SetGoroutineLabels(labels)

    memlimit.SetMemoryLimit()
    premain.StartMain()

    // ... rest of initialization
}
```

### Files to Modify

**Update each component's app.go:**
- `central/app/app.go` - label: "central"
- `sensor/kubernetes/app/app.go` - label: "kubernetes-sensor"
- `sensor/admission-control/app/app.go` - label: "admission-control"
- `config-controller/app/app.go` - label: "config-controller"
- `migrator/app/app.go` - label: "migrator"
- `compliance/cmd/compliance/app/app.go` - label: "compliance"
- `compliance/virtualmachines/roxagent/app/app.go` - label: "roxagent"
- `sensor/upgrader/app/app.go` - label: "sensor-upgrader"
- `roxctl/app/app.go` - label: "roxctl"

### Alternative: Dynamic Component Detection

Extract component name from `os.Args[0]`:

```go
// pkg/profiling/component.go
package profiling

import (
    "context"
    "os"
    "path/filepath"
    "runtime/pprof"
)

// SetComponentLabel tags the current goroutine with the component name
// derived from os.Args[0]. Call this early in app initialization.
func SetComponentLabel() {
    componentName := filepath.Base(os.Args[0])
    labels := pprof.Labels("component", componentName)
    pprof.SetGoroutineLabels(labels)
}
```

Then in each app.go:

```go
import "github.com/stackrox/rox/pkg/profiling"

func Run() {
    profiling.SetComponentLabel() // Automatically detects from os.Args[0]

    memlimit.SetMemoryLimit()
    premain.StartMain()
    // ...
}
```

### Verification

After implementation, heap profiles will show:

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

### Benefits

1. **Heap profiles** clearly identify which component they're from
2. **CPU profiles** also get component labels
3. **pprof queries** can filter by component
4. **Zero runtime overhead** - labels are metadata, not allocations
5. **Works with all profiling tools** - pprof, continuous profiling, etc.

### Testing

```bash
# Deploy with labels
# Capture heap profile from sensor
curl http://sensor-pod:6060/debug/pprof/heap > sensor-heap.pb.gz

# Verify component label
go tool pprof -raw sensor-heap.pb.gz | grep "component="
# Should show: component=kubernetes-sensor
```

## Implementation Effort

**Option 1 (Helper function):** 1-2 hours
- Create `pkg/profiling/component.go` with `SetComponentLabel()`
- Update 9 app.go files to call it
- Test with heap profiles from each component

**Option 2 (Manual labels):** 2-3 hours
- Hardcode component name in each app.go
- More explicit but more maintenance

**Recommendation:** Option 1 (helper function) - DRY, less maintenance
