# Conditional Initialization in Busybox Binary

## Overview

StackRox uses a busybox-style consolidated binary where multiple components (central, sensor, admission-control, config-controller, etc.) are built into a single binary and dispatched via os.Args[0].

## Problem

Go's package init() functions run when packages are imported, regardless of which component will actually execute. This caused all 535 init() functions to run for every component, leading to:

- Memory overhead (~166 MB for sensor, ~64 MB for central)
- OOMKills under race detector (config-controller: 7 restarts, admission-control: 6-7 restarts)

## Solution

Move initialization from package-level init() to explicit component-specific functions called from app.Run().

### Architecture

```
central/main.go (dispatcher)
    ↓ os.Args[0] check
component/app/app.go
    ↓ Run()
component/app/init.go
    ↓ initMetrics(), initCompliance(), etc.
```

### Migration Status

**Completed:**
- ✅ central/metrics (37 metrics)
- ✅ sensor/common/metrics (42 metrics)
- ✅ sensor/admission-control (6 metrics)

**Stubs (future work):**
- ⏳ compliance checks (109 files)
- ⏳ GraphQL loaders (15 files)

**Not migrated:**
- Shared infrastructure (logging, feature flags, env settings)
- Simple/negligible init() functions

## Usage

When adding new component-specific initialization:

1. Add function to `component/app/init.go`
2. Call from `component/app/app.go Run()` function
3. Avoid package-level init() for component-specific code

## Testing

Verify no cross-component init() execution:

```bash
# Check import chains
go list -f '{{.Deps}}' ./sensor/kubernetes | grep "central/"

# Should return no central/* packages (pkg/* is okay)
```

## References

- Design spec: docs/superpowers/specs/2026-04-13-conditional-init-design.md
- Implementation plan: docs/superpowers/plans/2026-04-13-conditional-init.md
- Verification report: docs/init-migration-verification.md
- ROX-33958: BusyBox binary consolidation OOMKill fixes
