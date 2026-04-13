# Conditional Init() Execution for BusyBox-Style Binary

## Context

**Problem:** PR #19819 (merged April 8, 2024) consolidated all StackRox binaries into a single busybox-style binary. This causes ALL package init() functions to execute regardless of which component runs (central, sensor, admission-control, config-controller, etc.).

**Impact:** Under the race detector (~10x memory multiplier), components with tight memory limits experience OOMKills:
- **config-controller** (128 Mi limit): 7 OOMKills, heap grew 61 MB → 227 MB
- **admission-control** (500 Mi limit): 6-7 OOMKills per replica, similar memory growth
- **Root cause:** All 535 init() functions run for every component, including:
  - 45+ central prometheus metrics (central-only)
  - 50+ sensor prometheus metrics (sensor-only)
  - 109 compliance check registrations (central-only)
  - 15+ GraphQL loader registrations (central-only)

**Goal:** Make initialization conditional based on which component is running. Components should only run their own init() logic, not all 535 init() functions.

**Approach:** Hybrid migration - move high-impact init() logic (~160 files) from package-level init() functions into explicit component-specific initialization functions called from app.Run().

## Architecture

**Three-Layer Model:**
```
main.go (dispatcher)
  → routes to component via os.Args[0]

component/app/app.go
  → Run() calls component-specific init functions

component/app/init.go (NEW)
  → initMetrics(), initCompliance(), initGraphQL(), etc.
  → replaces package-level init() functions
```

**Key Principle:** Move from implicit (package init()) to explicit (component initialization functions).

## Implementation Plan

### Phase 1: Infrastructure Setup (app/ packages)

**Goal:** Ensure all components have app/ package structure.

**Tasks:**
1. Verify which components already have app/ from PR #19819
2. Create missing app/ packages for components that need them:
   - `central/app/app.go` + `central/app/init.go`
   - `sensor/kubernetes/app/app.go` + `sensor/kubernetes/app/init.go`
   - `sensor/admission-control/app/app.go` + `sensor/admission-control/app/init.go`
   - `config-controller/app/app.go` + `config-controller/app/init.go`
   - Verify: `migrator/app/`, `compliance/cmd/compliance/app/`
3. Move existing main() logic to app.Run() where needed
4. Verify central/main.go dispatcher calls all app.Run() functions

**Files to modify:**
- New: `central/app/app.go`, `central/app/init.go`
- New: `sensor/kubernetes/app/init.go` (app.go may exist)
- New: `sensor/admission-control/app/init.go` (app.go may exist)
- New: `config-controller/app/init.go` (app.go may exist)
- Verify: `central/main.go` (dispatcher should already exist from PR #19819)

**Verification:**
- All components build successfully
- Dispatcher routing still works (os.Args[0] check)
- No behavior changes yet (this is structure-only)

### Phase 2: Critical Path (OOMKill Fixes)

**Goal:** Fix OOMKills in config-controller and admission-control.

**Priority Order:**
1. **config-controller** (128 Mi limit, 7 restarts)
   - Create init.go with minimal init functions
   - Break import chains to central packages

2. **admission-control** (500 Mi limit, 6-7 restarts)
   - Create init.go with initMetrics()
   - Move `sensor/admission-control/manager/metrics.go` init() logic

3. **Break sensor → central import chains**
   - Ensure sensor components don't import:
     - `central/compliance/checks`
     - `central/graphql`
     - `central/metrics`
   - This prevents sensor from loading central's heavy init() functions

**Files to modify:**
- New: `config-controller/app/init.go`
- New: `sensor/admission-control/app/init.go`
- Modify: `sensor/admission-control/app/app.go` (call initMetrics())
- Remove init() from: `sensor/admission-control/manager/metrics.go`

**Critical Success Metric:** Zero OOMKills in config-controller and admission-control after this phase.

### Phase 3: Central Initialization Migration

**Goal:** Migrate central's high-impact init() functions to explicit initialization.

**Target Init Functions (160 files):**

1. **Prometheus metrics** (28+ central metric files)
   - Primary: `central/metrics/init.go` (45+ metrics)
   - Others: compliance, debug, scanner definitions, detection, etc.
   - Pattern: Move prometheus.MustRegister() calls to initMetrics()

2. **Compliance checks** (109 files)
   - `central/compliance/checks/remote/all.go`
   - `central/compliance/checks/nist80053/*.go` (20+ files)
   - `central/compliance/checks/pcidss32/*.go` (25+ files)
   - `central/compliance/checks/hipaa_164/*.go` (15+ files)
   - `pkg/compliance/checks/kubernetes/*.go` (32 files)
   - Pattern: Consolidate framework.MustRegisterChecks() into initCompliance()

3. **GraphQL loaders** (15+ files)
   - `central/graphql/resolvers/loaders/*.go`
   - Files: policies.go, deployments.go, images.go, namespaces.go, nodes.go, etc.
   - Pattern: Move RegisterTypeFactory() calls to initGraphQL()

4. **Compliance standards** (5 files)
   - `central/compliance/standards/metadata/*.go`
   - Files: cis_kubernetes.go, hipaa_164.go, nist_800_53.go, nist_800_190.go, pci_dss_3_2.go
   - Pattern: Move AllStandards append logic to initComplianceStandards()

**Implementation:**

Create `central/app/init.go`:
```go
package app

func initMetrics() {
    // Move code from central/metrics/init.go
    prometheus.MustRegister(/* 45+ central metrics */)
}

func initCompliance() {
    // Consolidate 109 compliance check registrations
    framework.MustRegisterChecks(/* all checks */)
}

func initGraphQL() {
    // Move 15+ loader registrations
    RegisterTypeFactory(/* loaders */)
}

func initComplianceStandards() {
    // Move compliance standard metadata
}
```

Modify `central/app/app.go`:
```go
func Run() {
    memlimit.SetMemoryLimit()
    premain.StartMain()

    // NEW: Explicit initialization
    initMetrics()
    initCompliance()
    initGraphQL()
    initComplianceStandards()

    // ... existing central startup logic
}
```

**Files to modify:**
- New: `central/app/init.go` (consolidates 160+ init functions)
- Modify: `central/app/app.go` (add init function calls)
- Remove init() from: 28+ metric files, 109 compliance check files, 15+ loader files, 5 standard files

### Phase 4: Sensor Initialization Migration

**Goal:** Migrate sensor's init() functions to explicit initialization.

**Target Init Functions:**

1. **Prometheus metrics** (11+ sensor metric files)
   - Primary: `sensor/common/metrics/init.go` (50+ metrics)
   - Others: detector, pubsub, networkflow, centralproxy, VM metrics, etc.
   - Pattern: Move prometheus.MustRegister() calls to initMetrics()

**Implementation:**

Create `sensor/kubernetes/app/init.go`:
```go
package app

func initMetrics() {
    // Move code from sensor/common/metrics/init.go
    prometheus.MustRegister(/* 50+ sensor metrics */)
}
```

Modify `sensor/kubernetes/app/app.go`:
```go
func Run() {
    memlimit.SetMemoryLimit()
    premain.StartMain()

    // NEW: Explicit initialization
    initMetrics()

    // ... existing sensor startup logic
}
```

**Files to modify:**
- Modify: `sensor/kubernetes/app/init.go` (add initMetrics())
- Modify: `sensor/kubernetes/app/app.go` (call initMetrics())
- Remove init() from: `sensor/common/metrics/init.go` and 10+ other sensor metric files

### Phase 5: Low-Hanging Fruit Migration

**Goal:** Opportunistically migrate remaining easy-to-move init() functions.

**Target Categories:**

1. **Simple Registry Registrations**
   - `pkg/booleanpolicy/violationmessages/printer/gen-registrations.go` (100+ printer registrations)
   - `pkg/search/enumregistry/enum_registry.go` (enum map initialization)
   - Scanner-specific inits (scanner/enricher/nvd/nvd.go, etc.)

2. **Large Static Data Initialization**
   - `pkg/search/options.go` (37KB file, large map initialization)
   - `central/alert/mappings/options.go` (builds OptionsMap)

3. **Component-Specific Package Inits**
   - Operator scheme registrations
   - Roxctl-specific inits

**Approach:** Migrate these over time as we touch related code, or batch when convenient.

**Expected Coverage:**
- Phases 2-4: ~160 high-impact init() functions (fixes OOMKills)
- Phase 5: Additional ~40-90 functions (further optimization)
- Total migrated: ~200-250 of 535 init() functions

**Remaining:** ~300 init() functions are either truly shared, negligible impact, or have complex dependencies (defer to future work).

## Migration Pattern

**Before** (package-level init):
```go
// central/metrics/init.go
package metrics

var AlertProcessingDuration = prometheus.NewHistogramVec(...)

func init() {
    prometheus.MustRegister(AlertProcessingDuration)
}
```

**After** (explicit initialization):
```go
// central/metrics/metrics.go
package metrics

var AlertProcessingDuration = prometheus.NewHistogramVec(...)
// No init() function

// central/app/init.go
package app

func initMetrics() {
    prometheus.MustRegister(metrics.AlertProcessingDuration)
}
```

## Critical Files

**Dispatcher:**
- `central/main.go` - busybox dispatcher (verify only, should be from PR #19819)

**App packages to create/verify:**
- `central/app/app.go` + `central/app/init.go`
- `sensor/kubernetes/app/app.go` + `sensor/kubernetes/app/init.go`
- `sensor/admission-control/app/app.go` + `sensor/admission-control/app/init.go`
- `config-controller/app/app.go` + `config-controller/app/init.go`

**High-impact init() files to migrate (~160 files):**
- Central metrics: `central/metrics/init.go` + 27 others
- Sensor metrics: `sensor/common/metrics/init.go` + 10 others
- Compliance checks: 109 files in `central/compliance/checks/` and `pkg/compliance/checks/`
- GraphQL loaders: 15+ files in `central/graphql/resolvers/loaders/`
- Compliance standards: 5 files in `central/compliance/standards/metadata/`

## Verification

**CI Testing:**
- Build all components successfully
- Run existing test suites
- Deploy to test cluster with race detector enabled
- Monitor for OOMKills in config-controller and admission-control

**Expected Memory Impact:**

| Component | Current (race) | Target (race) | OOMKills Before | OOMKills After |
|-----------|---------------|---------------|-----------------|----------------|
| config-controller | ~150 MB (OOM @ 128 Mi) | < 100 MB | 7 | 0 |
| admission-control | ~600 MB (OOM @ 500 Mi) | < 400 MB | 6-7 per replica | 0 |
| central | 224 MB | ~224 MB (unchanged) | 0 | 0 |
| sensor | 227 MB | ~100 MB | 0 | 0 |

**Success Criteria:**
- Phase 2: Zero OOMKills in config-controller and admission-control
- Phase 3-4: Memory usage returns to pre-busybox levels for all components
- All phases: No functional regressions, all tests pass

## Rollout Strategy

**Merge Strategy:** Each phase merges independently
- Phase 1: Infrastructure, no behavior change, low risk
- Phase 2: Critical OOMKill fixes, high priority, merge ASAP
- Phase 3-4: Optimizations, merge after validation
- Phase 5: Opportunistic, merge when convenient

**Validation Between Phases:**
1. Merge to master
2. Wait for nightly build
3. Monitor admission-control/config-controller restart counts
4. Verify memory profiles
5. Proceed to next phase after validation

**Rollback Plan:** Changes are isolated to app/init.go files - can revert individual init functions without reverting entire change.
