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

### Critical Global Setup (7 files)
1. `pkg/logging/logging.go` - Logger initialization (must run first)
2. `pkg/grpc/codec.go` - gRPC codec registration (must run before any gRPC use)
3. `pkg/grpc/server.go` - Enables gRPC handling time histogram
4. `pkg/clientconn/useragent.go` - Sets default user agent
5. `pkg/httputil/proxy/proxy.go` - Initializes proxy transport
6. `pkg/cloudproviders/aws/certs.go` - Parses embedded AWS certificates
7. `pkg/mtls/crypto.go` - Sets cfssl log level (global side effect)

### Dev-Only (Build Tag Gated) (3 files)
8. `pkg/sync/deadlock_detect_dev.go` - Deadlock detection (only in dev builds)
9. `pkg/sync/mutex_dev.go` - Dev mutex instrumentation (only in dev builds)
10. `pkg/devbuild/init.go` - Dev build flag (only in dev builds)

These are safe to keep as `init()` since they either:
- Are fundamental (like logging) and needed by everything
- Register global settings that must happen before use (like gRPC codec)
- Set global library defaults that must apply everywhere (like cfssl log level)
- Are build-tag gated and only run in dev builds

## What Else Was Migrated (Additional 9 files)

### Static Data/Registry Initialization → Individual Init() functions

All static data initialization and registry setup were converted from `init()` to `Init()`:
- `pkg/administration/events/stream/stream.go` → `stream.Init()`
- `pkg/booleanpolicy/violationmessages/printer/gen-registrations.go` → `printer.Init()` (45 printer registrations)
- `pkg/gjson/modifiers.go` → `gjson.Init()` (GJSON custom modifiers)
- `pkg/net/internal/ipcheck/ipcheck.go` → `pkgnet.Init()` (via public wrapper in pkg/net/init.go)
- `pkg/search/enumregistry/enum_registry.go` → `enumregistry.Init()`
- `pkg/search/options.go` → `pkgsearch.Init()` (derived field maps)
- `pkg/renderer/kubernetes.go` → `renderer.Init()`
- `pkg/signatures/cosign_sig_fetcher.go` → `signatures.Init()`
- `pkg/tlsprofile/profile.go` → `tlsprofile.Init()`

**How to use:** Call each package's `Init()` function early in your component's `main()` function:
- **Central**: Calls all of the above from `central/app/init.go:initComponentLogic()`
- **Roxctl**: Calls `renderer.Init()` from `roxctl/app/init.go:initComponentLogic()`
- **Sensor**: Calls `pkgnet.Init()` from `sensor/kubernetes/app/app.go:Run()`

## Migration Status

- **Migrated:** 29 init() functions (5 metrics + 15 volume converters + 9 static data/registry)
- **Kept:** 10 init() functions (justified below)
- **Remaining:** 0 non-justified init() functions ✅

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
