# pkg/ init() Migration Guide

This document describes the migration of `init()` functions in `pkg/` to explicit initialization.

## Why

The busybox binary consolidation causes all `init()` functions to run for all components. Since `pkg/` is a shared library used by all components, we need explicit initialization to avoid unnecessary setup in binaries that don't need it.

## What Was Migrated

### Metrics (5 files) → Individual Init() functions

All Prometheus metric registrations were converted from `init()` to `Init()`:
- `pkg/grpc/metrics/prom.go` → `grpcmetrics.Init()`
- `pkg/images/metrics.go` → `images.Init()`
- `pkg/postgres/metrics.go` → `postgres.Init()`
- `pkg/rate/metrics.go` → `rate.Init()`
- `pkg/tlscheckcache/tlscheckcache_metrics.go` → `tlscheckcache.Init()`

**How to use:** Call each package's `Init()` function early in your component's `main()` function if you use metrics from that package. Note: These cannot be consolidated into a single `pkg/metrics/Init()` due to import cycles.

### Volume Converters (15 files) → pkg/protoconv/resources/volumes/RegisterAll()

All Kubernetes volume type converters were consolidated into one registration function:
- azure_disk, azure_file, cephfs, cinder, config, ebs, emptydir, gcepersistent, git, gluster, hostpath, nfs, persistent, rbd, secret

**How to use:** Call `volumes.RegisterAll()` before using protoconv for Kubernetes resources (typically in Central or Sensor initialization).

## What Kept init()

These files retain their `init()` functions - they are truly global or dev-only:

### Critical Global Setup (6 files)
1. `pkg/logging/logging.go` - Logger initialization (must run first)
2. `pkg/grpc/codec.go` - gRPC codec registration (must run before any gRPC use)
3. `pkg/grpc/server.go` - Enables gRPC handling time histogram
4. `pkg/clientconn/useragent.go` - Sets default user agent
5. `pkg/httputil/proxy/proxy.go` - Initializes proxy transport
6. `pkg/cloudproviders/aws/certs.go` - Parses embedded AWS certificates

### Dev-Only (Build Tag Gated) (3 files)
7. `pkg/sync/deadlock_detect_dev.go` - Deadlock detection (only in dev builds)
8. `pkg/sync/mutex_dev.go` - Dev mutex instrumentation (only in dev builds)
9. `pkg/devbuild/init.go` - Dev build flag (only in dev builds)

These are safe to keep as `init()` since they either:
- Are fundamental (like logging) and needed by everything
- Register global settings that must happen before use (like gRPC codec)
- Are build-tag gated and only run in dev builds

## Migration Status

- **Migrated:** 20 init() functions (5 metrics + 15 volume converters)
- **Kept:** 9 init() functions (justified above)
- **Remaining:** ~12 init() functions (lower priority, case-by-case evaluation needed)

## Example Usage

### Central
```go
import (
    grpcmetrics "github.com/stackrox/rox/pkg/grpc/metrics"
    "github.com/stackrox/rox/pkg/images"
    "github.com/stackrox/rox/pkg/postgres"
    "github.com/stackrox/rox/pkg/protoconv/resources/volumes"
    "github.com/stackrox/rox/pkg/rate"
    "github.com/stackrox/rox/pkg/tlscheckcache"
)

func main() {
    // Metrics initialization (call only what you use)
    grpcmetrics.Init()
    images.Init()
    postgres.Init()
    rate.Init()
    tlscheckcache.Init()

    // Volume converters (if using protoconv for k8s resources)
    volumes.RegisterAll()

    // ... rest of central initialization
}
```

### Sensor
```go
import (
    grpcmetrics "github.com/stackrox/rox/pkg/grpc/metrics"
    "github.com/stackrox/rox/pkg/protoconv/resources/volumes"
)

func main() {
    // Metrics initialization (sensor uses fewer metrics)
    grpcmetrics.Init()

    // Volume converters
    volumes.RegisterAll()

    // ... rest of sensor initialization
}
```

### Roxctl (CLI)
```go
func main() {
    // No need to call metric Init() or volumes.RegisterAll()
    // unless the CLI specifically uses those features

    // ... CLI logic
}
```
