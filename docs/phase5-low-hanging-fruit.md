# Phase 5: Low-Hanging Fruit Migration Plan

> **⚠️ IMPORTANT:** This document includes scanner/* analysis. Scanner is NOT part of the busybox binary (separate service). For busybox-scoped recommendations, see `phase5-busybox-scope.md`.

## Executive Summary

**Total init() functions surveyed:** ~326 (535 total - 85 already migrated - 124 stubbed)

**Analyzed by 6-agent team:**
- booleanpolicy-analyzer: 1 init (44 registrations) - **2/5 stars** - Skip
- roxctl-analyzer: 2 inits - **2/5 stars** - Skip
- scanner-analyzer: 5 inits - **⭐⭐⭐⭐⭐** - **TOP PRIORITY**
- operator-analyzer: 6 inits - **1/5 stars** - Skip
- search-analyzer: 2 inits - **1/5 stars** - Skip
- pkg-analyzer: 70 inits - **Mixed ratings** - Several good candidates

**Recommended for Phase 5:** 27 init() functions across 8 categories

---

## Priority Tier 1: Slam Dunks (5★ - Do First)

### 1. Scanner Component Isolation (5 init functions)
**Location:** `scanner/*`
**Rating:** ⭐⭐⭐⭐⭐
**Effort:** 2-4 hours
**Impact:** Complete scanner isolation from init() overhead

All 5 scanner init() functions are isolated (zero cross-component imports) and 4/5 are trivial:

1. **scanner/cmd/scanner/main.go** - Move proxy + memlimit to top of main()
   - Complexity: Trivial - just move 2 lines

2. **scanner/enricher/nvd/nvd.go** - URL.Parse on constant
   - Replace: `var defaultFeed = must(url.Parse(DefaultFeeds))`

3. **scanner/enricher/csaf/internal/zreader/zreader.go** - Max header size calculation
   - Replace with computed const or helper function

4. **scanner/indexer/indexer.go** - LRU cache creation
   - Replace: `var regsNoRange = must(lru.New[string, struct{}](100))`

5. **scanner/matcher/matcher.go** - Node.js matcher registration
   - Medium effort: Move to explicit `RegisterMatchers()` called at startup

**Files:** 5
**Expected time:** 2-4 hours
**Benefits:** Scanner completely init()-free

### 2. Volume Type Registry (15 init functions)
**Location:** `pkg/protoconv/resources/volumes/*.go`
**Rating:** ⭐⭐⭐⭐⭐
**Effort:** 1-2 hours
**Impact:** Clean registry pattern, -15 init() functions

Pattern: 15 files each do `VolumeRegistry[type] = createFunc` in init()

**Migration:**
```go
// New file: pkg/protoconv/resources/volumes/registry.go
func RegisterAll() {
    VolumeRegistry = map[string]func(interface{}) VolumeSource{
        azureDiskType: createAzureDisk,
        nfsType: createNFS,
        ebsType: createEBS,
        // ... all 15 entries
    }
}
```

Call `RegisterAll()` from first use or component init.

**Files:** 15
**Expected time:** 1-2 hours
**Benefits:** Clean, testable, -15 init() functions

---

## Priority Tier 2: Easy Wins (4★ - Quick Hits)

### 3. Client Connection Default User Agent
**Location:** `pkg/clientconn/useragent.go`
**Rating:** ⭐⭐⭐⭐
**Effort:** 30 minutes
**Impact:** Remove unnecessary default init

Current pattern: init() sets default, then every component overrides it anyway.

**Migration:** Remove init(), require explicit `SetUserAgent()` call. All components already do this (roxctl, sensor, etc.).

**Files:** 1
**Expected time:** 30 minutes

### 4. Central Renderer Asset Registration
**Location:** `pkg/renderer/kubernetes.go`
**Rating:** ⭐⭐⭐⭐
**Effort:** 30 minutes
**Impact:** Move Central-only code to Central init

Central-specific asset registration currently in pkg/.

**Migration:** Move `assetFileNameMap.AddWithName()` call to `central/app/init.go`.

**Files:** 1
**Expected time:** 30 minutes

### 5. Dead Code Deletion
**Location:** `pkg/sync/deadlock_detect_dev.go`
**Rating:** ⭐⭐⭐⭐
**Effort:** 5 minutes
**Impact:** Delete unreachable code

Build tag `!go1.17` means this never compiles with modern Go. Safe to delete.

**Files:** 1
**Expected time:** 5 minutes

---

## Priority Tier 3: Good ROI (3★ - Solid Improvements)

### 6. PostgreSQL Metrics (Central-only)
**Location:** `pkg/postgres/metrics.go`
**Rating:** ⭐⭐⭐
**Effort:** 30 minutes
**Impact:** Move Central-specific metrics to Central init

Uses `CentralSubsystem` - clearly Central-only, but in pkg/.

**Migration:** Move prometheus.MustRegister() to `central/app/init.go` initMetrics().

**Files:** 1
**Expected time:** 30 minutes

### 7. GJSON Custom Modifiers
**Location:** `pkg/gjson/modifiers.go`
**Rating:** ⭐⭐⭐
**Effort:** 1 hour
**Impact:** Lazy registration, only load if needed

Registers custom GJSON modifiers into library global.

**Migration:** Wrap in `sync.Once`, call before first GJSON use (limited code paths).

**Files:** 1
**Expected time:** 1 hour

### 8. Utility Data Initialization (7 files)
**Locations:** Various pkg/* utilities
**Rating:** ⭐⭐⭐
**Effort:** 2-3 hours total
**Impact:** Convert to lazy or package-level var patterns

- `pkg/tlsprofile/profile.go` - Build TLS version/cipher maps from stdlib
- `pkg/net/internal/ipcheck/ipcheck.go` - Precompute IPv4 masks
- `pkg/images/enricher/metadata.go` - Precompute empty metadata hash
- `pkg/cloudproviders/aws/certs.go` - Parse AWS PEM certs
- `pkg/httputil/proxy/proxy.go` - Clone default transport
- `pkg/signatures/cosign_sig_fetcher.go` - Clone transport with insecure TLS

**Migration:** Convert to `sync.Once` lazy init or `var = func(){}()` patterns.

**Files:** 7
**Expected time:** 2-3 hours total

---

## Priority Tier 4: Larger Refactors (3★ - Consider for Future)

### 9. Compliance Check Registrations (32 files)
**Location:** `pkg/compliance/checks/*`
**Rating:** ⭐⭐⭐
**Effort:** 4-6 hours (mechanical but large)
**Impact:** Already documented in compliance-check-init-migration.md

32 files each call `standards.RegisterChecksForStandard()` in init().

**Note:** This overlaps with the 109-file compliance migration already documented. May want to combine efforts.

**Files:** 32
**Expected time:** 4-6 hours
**Decision:** Consider as part of larger compliance refactor

---

## Skip List (Not Worth Migrating)

### Foundational / Framework Code (1★)
- `pkg/logging/logging.go` - Root logger, must run first
- `pkg/grpc/codec.go` - Codec registration, ordering-critical
- `pkg/grpc/server.go` - gRPC prometheus/logging, ordering-critical
- `pkg/mtls/crypto.go` - Simple log level adjustment, harmless
- `operator/cmd/main.go` + CRD types - kubebuilder conventions

### Negligible Overhead (1-2★)
- `pkg/search/options.go` - ~50KB, deeply embedded
- `pkg/search/enumregistry/enum_registry.go` - Empty map init
- `pkg/booleanpolicy/violationmessages/printer/gen-registrations.go` - Code-generated
- `roxctl/common/flags/imageFlavor.go` - Bool check + string assign
- `roxctl/helm/internal/common/chartnames.go` - String concatenation

### Build-Tag Scoped (2★)
- `pkg/devbuild/init.go` - Development safety check
- `pkg/sync/mutex_dev.go` - Dev-only lock timeout

---

## Recommended Phase 5 Execution Plan

### Week 1: Tier 1 (Slam Dunks)
1. **Scanner isolation** (5 inits) - 2-4 hours
2. **Volume registry** (15 inits) - 1-2 hours

**Total:** 20 init() functions eliminated, ~4-6 hours effort

### Week 2: Tier 2 (Easy Wins)
3. **clientconn useragent** (1 init) - 30 min
4. **renderer asset** (1 init) - 30 min
5. **Dead code deletion** (1 init) - 5 min

**Total:** 3 init() functions eliminated, ~1 hour effort

### Week 3: Tier 3 (Good ROI)
6. **postgres metrics** (1 init) - 30 min
7. **gjson modifiers** (1 init) - 1 hour
8. **Utility data** (7 inits) - 2-3 hours

**Total:** 9 init() functions eliminated, ~3-4 hours effort

### Optional: Tier 4 (Future Work)
9. **Compliance checks** (32 inits) - Combine with main compliance refactor

---

## Success Metrics

**Phase 5 Target:** Eliminate 27-32 additional init() functions

**Current Status:**
- Migrated: 85 (Phase 1-4)
- Stubbed: 124 (compliance + GraphQL, future work)
- **Phase 5 candidates: 27-32**
- Skip: ~180-200 (foundational, negligible, or framework code)

**Post-Phase 5 Status:**
- **Total migrated: 112-117 of 535 (21-22%)**
- **Documented for future: 124 (23%)**
- **Remaining: ~294-299 (55%)** - mostly foundational, build-scoped, or negligible

**Key Wins:**
- ✅ Scanner completely init()-free
- ✅ Volume registry pattern cleaned up
- ✅ Central-specific code properly isolated
- ✅ Dead code removed

---

## Implementation Notes

### Testing Strategy
- Build all components after each migration
- Run unit tests for affected packages
- Verify no init() ordering dependencies broken
- Test with race detector enabled

### Commit Strategy
- One commit per tier (logical grouping)
- Clear commit messages with before/after init() count
- Reference ROX-33958 in all commits

### Rollback Plan
- Each tier is independent
- Easy to revert individual commits
- No cross-tier dependencies

---

## Appendix: Full Analysis by Component

See individual agent reports for detailed analysis:
- booleanpolicy: Code-generated, 44 registrations, shared - Skip
- roxctl: 2 trivial inits, negligible overhead - Skip
- scanner: 5 isolated inits, 4 trivial - **TOP PRIORITY**
- operator: kubebuilder patterns - Skip
- search: ~50KB overhead, embedded - Skip
- pkg: 70 inits, mixed - 27 candidates identified above

**Total time investment for Phase 5:** ~8-11 hours
**Total init() functions eliminated:** 27-32
**Impact:** Improved isolation, cleaner architecture, modest memory savings

---

## ⚠️ Scanner Out of Scope

**Scanner is a separate binary**, not part of the busybox consolidation (not in central/main.go dispatcher).

**Scanner init() functions (5 total, 4/5 trivial):**
- scanner/cmd/scanner/main.go
- scanner/enricher/nvd/nvd.go
- scanner/enricher/csaf/internal/zreader/zreader.go
- scanner/indexer/indexer.go
- scanner/matcher/matcher.go

**Recommendation:** Address in separate PR for scanner-specific optimization (2-4 hours effort).

**For busybox-scoped Phase 5 recommendations, see:** `docs/phase5-busybox-scope.md`
