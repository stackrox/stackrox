# Implementation Plan: isPublic() Optimization

## Overview

Optimize the `isPublic()` function in `pkg/net/addr.go` using a phased approach:
- **Phase 1**: Scalar optimization (Go 1.25) - optimize current implementation
- **Phase 2**: SIMD enablement (Go 1.26+) - leverage experimental SIMD instructions

## Current State

- **Go Version**: 1.25.0
- **Current Performance**: ~33ns per public IPv4 check
- **Architecture**: Uses `net.IPNet.Contains()` in a loop (5 iterations for IPv4, 7 for IPv6)
- **Issue**: Performance bottleneck in network flow manager's `IsExternal()` method

---

## Phase 1: Scalar Optimization (Go 1.25)

### Goal
Improve performance by 3-5x using optimized scalar implementation without requiring Go 1.26.

### Changes Required

#### 1. Optimize `pkg/net/addr.go` Scalar Functions

**Current Problem**:
- `isPublicIPv4Scalar()` (line 69) loops through `netutil.IPv4PrivateNetworks` and calls `privateIPNet.Contains(ip)`
- `net.IPNet.Contains()` has significant overhead (creates slices, method calls, generic network checking)
- This is the performance bottleneck even on non-SIMD platforms

**Solution**: Replace with optimized bitwise operations
```go
func isPublicIPv4Scalar(ip net.IP) bool {
    // Convert to IPv4 bytes
    ipv4 := ip.To4()
    if ipv4 == nil {
        return false
    }

    // Convert to uint32 for efficient bitwise operations
    ipInt := uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3])

    // Check each private network with bitwise mask-and-compare
    // 10.0.0.0/8
    if (ipInt & 0xFF000000) == 0x0A000000 {
        return false
    }
    // 172.16.0.0/12
    if (ipInt & 0xFFF00000) == 0xAC100000 {
        return false
    }
    // 192.168.0.0/16
    if (ipInt & 0xFFFF0000) == 0xC0A80000 {
        return false
    }
    // 100.64.0.0/10
    if (ipInt & 0xFFC00000) == 0x64400000 {
        return false
    }
    // 169.254.0.0/16
    if (ipInt & 0xFFFF0000) == 0xA9FE0000 {
        return false
    }

    return true // Is public
}
```

**Why**:
- Eliminates `net.IPNet.Contains()` overhead
- Uses CPU-friendly bitwise operations
- Provides 3-5x speedup immediately on Go 1.25
- No external dependencies or experimental features required

#### 2. Optional: Refactor into `pkg/net/internal/simdutil/` package

**Note**: This step is optional for Phase 1. You can optimize the scalar functions directly in `addr.go`, OR extract them to a separate `simdutil` package for better organization. The `simdutil` package approach is recommended if you plan to proceed to Phase 2 (SIMD), as it prepares the code structure.

**If using simdutil approach**:

**File: `pkg/net/internal/simdutil/ipcheck_stub.go`** (build tag for non-SIMD)
```go
//go:build !amd64 || !goexperiment.simd

package simdutil

import "encoding/binary"

var (
    // Global constants - reused across all calls
    ipv4Masks = [5]uint32{
        0xFF000000, // 10.0.0.0/8
        0xFFFF0000, // 192.168.0.0/16
        0xFFF00000, // 172.16.0.0/12
        0xFFC00000, // 100.64.0.0/10
        0xFFFF0000, // 169.254.0.0/16
    }
    ipv4Prefixes = [5]uint32{
        0x0A000000, 0xC0A80000, 0xAC100000, 0x64400000, 0xA9FE0000,
    }
)

func CheckIPv4Public(d [4]byte) bool {
    ip := binary.BigEndian.Uint32(d[:])

    // Loop through all 5 private networks
    for i := 0; i < 5; i++ {
        if (ip & ipv4Masks[i]) == ipv4Prefixes[i] {
            return false
        }
    }

    return true
}
```

**Then update `addr.go`**:
```go
func isPublicIPv4Scalar(ip net.IP) bool {
    ipv4 := ip.To4()
    if ipv4 == nil {
        return false
    }
    var d [4]byte
    copy(d[:], ipv4)
    return simdutil.CheckIPv4Public(d)
}
```

#### 3. Update `pkg/net/addr.go` methods
```go
func (d ipv4data) isPublic() bool {
    return simdutil.CheckIPv4Public(d)
}
```

### Expected Performance (Phase 1)

Based on measurements with optimized scalar:
- Public IPv4: 33ns → **~10ns** (3.3x speedup)
- Private IPv4: 11-35ns → **~3-6ns** (3-5x speedup)
- Batch (1000 IPs): 27µs → **~8-10µs** (3x speedup)

### Testing Requirements (Phase 1)

1. **Unit Tests** (must pass):
   ```bash
   go test ./pkg/net/...
   go test ./pkg/net/internal/simdutil/...
   ```

2. **Benchmarks** (before/after comparison):
   ```bash
   # Baseline (save before changes)
   go test -bench=BenchmarkIsPublic -benchmem -count=10 ./pkg/net/ > baseline.txt

   # After optimization
   go test -bench=BenchmarkIsPublic -benchmem -count=10 ./pkg/net/ > optimized.txt

   # Compare
   benchstat baseline.txt optimized.txt
   ```

3. **Expected Results**:
   - ✅ All tests pass
   - ✅ 3x+ speedup for public IPs
   - ✅ 0 allocations/op (no regression)
   - ✅ No correctness issues

### Deployment Criteria (Phase 1)

- [ ] All unit tests pass
- [ ] Benchmarks show 3x+ improvement
- [ ] Zero memory allocations maintained
- [ ] Code review approved
- [ ] Integration tests pass

### Rollout (Phase 1)

1. Merge changes to `master`
2. Deploy with regular release cycle
3. Monitor production metrics:
   - Network flow enrichment latency
   - CPU utilization in network flow manager
   - No correctness issues in network graph

---

## Phase 2: SIMD Enablement (Go 1.26+)

### Prerequisites

- [ ] Go 1.26.0 or later released
- [ ] `simd/archsimd` API declared stable (check Go release notes)
- [ ] Phase 1 deployed and validated in production

### Goal
Achieve 10-12x total speedup using real SIMD vector instructions on AMD64.

### Changes Required

#### 1. Update `go.mod`
```go
module github.com/stackrox/rox

go 1.26.0
```

#### 2. Create build-tagged SIMD files

**File: `pkg/net/addr_simd_amd64.go`**
```go
//go:build amd64 && goexperiment.simd

package net

import (
    "net"
    "github.com/stackrox/rox/pkg/net/internal/simdutil"
)

func (d ipv4data) isPublic() bool {
    return simdutil.CheckIPv4Public(d)
}

func (d ipv6data) isPublic() bool {
    return isPublicIPv6Scalar(net.IP(d.bytes()))
}
```

**File: `pkg/net/addr_simd_stub.go`**
```go
//go:build !amd64 || !goexperiment.simd

package net

import "net"

func (d ipv4data) isPublic() bool {
    return isPublicIPv4Scalar(net.IP(d.bytes()))
}

func (d ipv6data) isPublic() bool {
    return isPublicIPv6Scalar(net.IP(d.bytes()))
}
```

#### 3. Update `simdutil` package

**Rename**: `ipcheck.go` → `ipcheck_stub.go`
**Add build tag**: `//go:build !amd64 || !goexperiment.simd`

**Create**: `ipcheck_amd64.go` with SIMD implementation:
```go
//go:build amd64 && goexperiment.simd

package simdutil

import (
    "encoding/binary"
    "simd/archsimd"
)

var (
    ipv4Masks4 = [4]uint32{
        0xFF000000, 0xFFFF0000, 0xFFF00000, 0xFFC00000,
    }
    ipv4Prefixes4 = [4]uint32{
        0x0A000000, 0xC0A80000, 0xAC100000, 0x64400000,
    }
    ipv4Mask5th   uint32 = 0xFFFF0000
    ipv4Prefix5th uint32 = 0xA9FE0000
)

func CheckIPv4Public(d [4]byte) bool {
    ip := binary.BigEndian.Uint32(d[:])

    // SIMD: Check 4 networks in parallel
    ipVec := archsimd.BroadcastUint32x4(ip)
    masks := archsimd.LoadUint32x4(&ipv4Masks4)
    prefixes := archsimd.LoadUint32x4(&ipv4Prefixes4)

    masked := ipVec.And(masks)
    matches := masked.Equal(prefixes)

    if matches.ToBits() != 0 {
        return false
    }

    // Check 5th network
    if (ip & ipv4Mask5th) == ipv4Prefix5th {
        return false
    }

    return true
}
```

#### 4. Add SIMD correctness tests

**File: `pkg/net/addr_simd_test.go`**
```go
//go:build amd64 && goexperiment.simd

package net

import (
    "net"
    "testing"
    "github.com/stackrox/rox/pkg/net/internal/simdutil"
)

// TestSIMDCorrectness compares SIMD vs scalar
func TestSIMDCorrectness(t *testing.T) {
    testIPs := []string{
        "8.8.8.8", "10.0.0.1", "172.16.0.1",
        "192.168.1.1", "100.64.0.1", // ... more IPs
    }

    for _, ipStr := range testIPs {
        addr := ParseIP(ipStr)
        simdResult := addr.IsPublic()
        scalarResult := isPublicIPv4Scalar(net.ParseIP(ipStr))

        if simdResult != scalarResult {
            t.Errorf("SIMD mismatch for %s: SIMD=%v, Scalar=%v",
                ipStr, simdResult, scalarResult)
        }
    }
}

// FuzzIsPublicSIMD ensures SIMD matches scalar for random IPs
func FuzzIsPublicSIMD(f *testing.F) {
    f.Fuzz(func(t *testing.T, a, b, c, d byte) {
        ip := net.IPv4(a, b, c, d)
        addr := FromNetIP(ip)

        simdResult := addr.IsPublic()
        scalarResult := isPublicIPv4Scalar(ip)

        if simdResult != scalarResult {
            t.Errorf("Mismatch for %v: SIMD=%v, Scalar=%v",
                ip, simdResult, scalarResult)
        }
    })
}
```

### Expected Performance (Phase 2)

Based on measured results with real SIMD:
- Public IPv4: 33ns → **2.8ns** (12x speedup)
- Private IPv4: 11-35ns → **2.9-3.9ns** (3.5-10x speedup)
- Batch (1000 IPs): 27µs → **4µs** (6.8x speedup)
- Worst case (1000 public): 32µs → **3.6µs** (9x speedup)

### Testing Requirements (Phase 2)

1. **Without SIMD** (backward compatibility):
   ```bash
   go test ./pkg/net/...  # Should use scalar fallback
   go test -bench=BenchmarkIsPublic ./pkg/net/
   ```

2. **With SIMD** (new functionality):
   ```bash
   GOEXPERIMENT=simd go test ./pkg/net/...
   GOEXPERIMENT=simd go test ./pkg/net/ -run SIMD
   GOEXPERIMENT=simd go test -bench=BenchmarkIsPublic -count=10 ./pkg/net/ > simd.txt
   ```

3. **Fuzz testing** (10+ minutes):
   ```bash
   GOEXPERIMENT=simd go test ./pkg/net/ -fuzz=FuzzIsPublicSIMD -fuzztime=10m
   ```

4. **Comparison**:
   ```bash
   benchstat baseline.txt simd.txt
   # Expected: 10-12x improvement for public IPs
   ```

### Deployment Criteria (Phase 2)

- [ ] Go 1.26 upgraded across infrastructure
- [ ] All tests pass (with and without SIMD)
- [ ] SIMD tests pass on AMD64
- [ ] Fuzz tests run for 10+ minutes with no failures
- [ ] Benchmarks show 10x+ improvement on AMD64
- [ ] No performance regression on ARM64 (uses scalar fallback)
- [ ] Code review approved

### Build Configuration

**Makefile addition (optional)**:
```makefile
.PHONY: bench-simd
bench-simd:
	@echo "Running baseline benchmarks..."
	go test -bench=BenchmarkIsPublic -benchmem -count=10 ./pkg/net/ > /tmp/baseline.txt
	@echo "Running SIMD benchmarks..."
	GOEXPERIMENT=simd go test -bench=BenchmarkIsPublic -benchmem -count=10 ./pkg/net/ > /tmp/simd.txt
	benchstat /tmp/baseline.txt /tmp/simd.txt
```

### Rollout Strategy (Phase 2)

1. **Testing Phase** (2-4 weeks):
   - Deploy with `GOEXPERIMENT=simd` to test cluster
   - Monitor performance metrics
   - Validate correctness in network flow graphs
   - No issues observed → proceed

2. **Canary Deployment**:
   - Deploy to 10% of sensors with SIMD enabled
   - Monitor for 1 week:
     - Network flow enrichment latency
     - CPU utilization
     - External vs internal classification accuracy
   - No issues → expand

3. **Full Rollout**:
   - Deploy to all AMD64 sensors
   - ARM64 sensors continue using optimized scalar (still 3x faster)

4. **Monitoring**:
   - Track metrics for 2-4 weeks
   - Collect performance data
   - Document improvements

---

## Success Metrics

### Phase 1 Success Criteria
- ✅ 3x+ speedup achieved
- ✅ Zero correctness regressions
- ✅ Zero memory overhead
- ✅ Production deployment successful

### Phase 2 Success Criteria
- ✅ 10x+ speedup on AMD64
- ✅ No regression on ARM64
- ✅ SIMD tests pass on AMD64
- ✅ Fuzz tests pass (10+ min)
- ✅ Production metrics improved

## Risks & Mitigations

### Phase 1 Risks
| Risk | Impact | Mitigation |
|------|--------|------------|
| Correctness bug | High | Comprehensive test suite, fuzz testing |
| Performance regression | Medium | Benchmark before/after, rollback plan |
| Build complexity | Low | Simple refactoring, no new dependencies |

### Phase 2 Risks
| Risk | Impact | Mitigation |
|------|--------|------------|
| SIMD API changes | Medium | Build tags isolate SIMD code, easy rollback |
| Platform-specific bugs | Medium | Comprehensive testing on AMD64 and ARM64 |
| Experimental flag instability | Low | Monitor Go 1.26 release notes, wait for stability |
| ARM64 regression | Low | Build tags ensure ARM64 uses proven scalar code |

## Timeline

### Phase 1: Scalar Optimization (Immediate)
- **Week 1**: Implementation and unit testing
- **Week 2**: Benchmarking and code review
- **Week 3**: Merge and deploy to staging
- **Week 4**: Production deployment
- **Week 5-6**: Monitoring and validation

### Phase 2: SIMD Enablement (After Go 1.26 release)
- **Pre-requisite**: Go 1.26 released and stable
- **Week 1**: Implement SIMD version
- **Week 2**: Testing and fuzz testing
- **Week 3**: Deploy to test cluster
- **Week 4-5**: Canary deployment (10% of sensors)
- **Week 6-7**: Full rollout
- **Week 8-10**: Monitoring and documentation

## Documentation Updates

After each phase, update:
- [x] `pkg/net/SIMD_README.md` - Implementation guide
- [x] `pkg/net/BENCHMARK_RESULTS.md` - Performance data
- [ ] Release notes - Mention performance improvements
- [ ] Architecture docs - Update network flow manager documentation

## References

- Original implementation: `pkg/net/addr.go` lines 67-75 (IPv4), 105-113 (IPv6)
- Network flow manager: `sensor/common/networkflow/manager/manager_impl.go:96`
- Go 1.26 SIMD docs: https://pkg.go.dev/simd/archsimd (when released)
- Private network definitions: `pkg/netutil/private_subnet.go`

---

## Quick Reference Commands

```bash
# Phase 1: Scalar optimization
go test ./pkg/net/...
go test -bench=BenchmarkIsPublic -count=10 ./pkg/net/

# Phase 2: SIMD testing
GOEXPERIMENT=simd go test ./pkg/net/...
GOEXPERIMENT=simd go test -bench=BenchmarkIsPublic -count=10 ./pkg/net/
GOEXPERIMENT=simd go test -fuzz=FuzzIsPublicSIMD -fuzztime=10m ./pkg/net/

# Comparison
benchstat baseline.txt optimized.txt
benchstat baseline.txt simd.txt
```
