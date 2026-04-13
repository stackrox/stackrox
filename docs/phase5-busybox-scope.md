# Phase 5: Low-Hanging Fruit (BusyBox Scope Only)

## Scope Clarification

**BusyBox Components (from central/main.go dispatcher):**
- central
- migrator
- compliance
- kubernetes-sensor
- sensor-upgrader
- admission-control
- config-controller
- roxagent
- roxctl

**Out of Scope:**
- **scanner** - Separate binary, NOT part of busybox consolidation
  - Note: Scanner has 5 easy-to-migrate init() functions (see separate note below)
  - Should be addressed in a separate PR (scanner-specific optimization)

---

## Revised Priority List (BusyBox Only)

### Priority Tier 1: Slam Dunks (5★)

#### 1. Volume Type Registry (15 init functions)
**Location:** `pkg/protoconv/resources/volumes/*.go`
**Rating:** ⭐⭐⭐⭐⭐
**Effort:** 1-2 hours
**BusyBox Usage:** Need to verify which components import this

15 files doing `VolumeRegistry[type] = createFunc` in init():
- azure_disk.go, nfs.go, ebs.go, gce_persistent_disk.go, etc.

**Verification needed:**
```bash
# Check which busybox components import volumes package
for comp in central sensor/kubernetes sensor/admission-control config-controller migrator compliance/cmd/compliance roxctl; do
  echo "=== $comp ==="
  go list -f '{{.Deps}}' ./$comp 2>/dev/null | grep "pkg/protoconv/resources/volumes" || echo "No import"
done
```

**Migration:** Single `RegisterAll()` function replacing 15 init() calls.

**Files:** 15
**Expected time:** 1-2 hours
**Impact:** Clean registry pattern if used by busybox

---

### Priority Tier 2: Easy Wins (4★)

#### 2. Client Connection Default User Agent
**Location:** `pkg/clientconn/useragent.go`
**Rating:** ⭐⭐⭐⭐
**Effort:** 30 minutes
**BusyBox Usage:** Yes - imported by roxctl, sensor, admission-control

Current: init() sets default "StackRox", then each component overrides.

**Migration:** Remove init(), require explicit `SetUserAgent()` call.

**Files:** 1
**Expected time:** 30 minutes

#### 3. Central Renderer Asset Registration
**Location:** `pkg/renderer/kubernetes.go`
**Rating:** ⭐⭐⭐⭐
**Effort:** 30 minutes
**BusyBox Usage:** Central-only

**Migration:** Move `assetFileNameMap.AddWithName()` to `central/app/init.go`.

**Files:** 1
**Expected time:** 30 minutes

#### 4. Dead Code Deletion
**Location:** `pkg/sync/deadlock_detect_dev.go`
**Rating:** ⭐⭐⭐⭐
**Effort:** 5 minutes
**BusyBox Usage:** N/A - dead code (build tag `!go1.17`)

Build tag ensures this never compiles with Go 1.17+. Safe to delete.

**Files:** 1
**Expected time:** 5 minutes

---

### Priority Tier 3: Good ROI (3★)

#### 5. PostgreSQL Metrics (Central-only)
**Location:** `pkg/postgres/metrics.go`
**Rating:** ⭐⭐⭐
**Effort:** 30 minutes
**BusyBox Usage:** Central-only (uses `CentralSubsystem`)

**Migration:** Move prometheus.MustRegister() to `central/app/init.go` initMetrics().

**Files:** 1
**Expected time:** 30 minutes

#### 6. GJSON Custom Modifiers
**Location:** `pkg/gjson/modifiers.go`
**Rating:** ⭐⭐⭐
**Effort:** 1 hour
**BusyBox Usage:** Need to verify usage in busybox components

**Verification needed:**
```bash
# Check which busybox components use gjson
grep -r "github.com/tidwall/gjson" central/ sensor/ config-controller/ migrator/ roxctl/ compliance/ | grep "import"
```

**Migration:** Wrap in `sync.Once`, call before first GJSON use.

**Files:** 1
**Expected time:** 1 hour

#### 7. Utility Data Initialization (subset)
**Locations:** Various pkg/* utilities
**Rating:** ⭐⭐⭐
**Effort:** Need to verify busybox usage first

**Candidates requiring verification:**
- `pkg/tlsprofile/profile.go` - TLS version/cipher maps
- `pkg/httputil/proxy/proxy.go` - Default transport clone
- `pkg/images/enricher/metadata.go` - Empty metadata hash
- `pkg/cloudproviders/aws/certs.go` - Parse AWS PEM certs
- `pkg/signatures/cosign_sig_fetcher.go` - Transport with insecure TLS

**Skip (internal/isolated):**
- `pkg/net/internal/ipcheck/ipcheck.go` - Internal package

**Expected time:** 1-2 hours for verified candidates

---

## Verification Required

Before implementing, verify which packages are actually imported by busybox components:

```bash
# Generate busybox component dependency report
for comp in central migrator compliance/cmd/compliance sensor/kubernetes sensor/upgrader sensor/admission-control config-controller compliance/virtualmachines/roxagent roxctl; do
  echo "=== $comp ==="
  go list -f '{{.Deps}}' ./$comp 2>/dev/null | grep "pkg/protoconv/resources/volumes\|pkg/gjson\|pkg/tlsprofile\|pkg/httputil/proxy\|pkg/images/enricher\|pkg/cloudproviders/aws\|pkg/signatures" || echo "None"
done
```

---

## Revised Estimate (BusyBox Scope)

**Tier 1 (5★):**
- Volume registry: 15 inits (if used by busybox)

**Tier 2 (4★):**
- clientconn useragent: 1 init
- renderer asset: 1 init
- dead code deletion: 1 init

**Tier 3 (3★):**
- postgres metrics: 1 init
- gjson modifiers: 1 init (if used)
- utility data: ~3-5 inits (subset verified for busybox)

**Conservative estimate:** 8-15 init() functions
**Optimistic estimate:** 20-25 init() functions (if all candidates used)
**Effort:** 4-8 hours

---

## Out of Scope (Separate PRs)

### Scanner Init Cleanup (5 functions)
**Rationale:** Scanner is a separate binary, not part of busybox
**Location:** `scanner/*`
**Effort:** 2-4 hours
**Rating:** ⭐⭐⭐⭐⭐ (all isolated, 4/5 trivial)

**Recommendation:** Create separate issue/PR for scanner init cleanup:
- scanner/cmd/scanner/main.go
- scanner/enricher/nvd/nvd.go
- scanner/enricher/csaf/internal/zreader/zreader.go
- scanner/indexer/indexer.go
- scanner/matcher/matcher.go

---

## Skip List (Not Worth Migrating for BusyBox)

**Foundational / Framework (1★):**
- pkg/logging/logging.go - Root logger, must run first
- pkg/grpc/codec.go, pkg/grpc/server.go - Ordering-critical
- pkg/mtls/crypto.go - Harmless log level adjustment

**Negligible Overhead (1-2★):**
- pkg/search/options.go - ~50KB, central-only usage
- pkg/search/enumregistry/enum_registry.go - Empty map init
- pkg/booleanpolicy/violationmessages/printer/gen-registrations.go - Code-generated, shared
- roxctl/* - Only 2 inits, negligible

**Build-Tag Scoped (2★):**
- pkg/devbuild/init.go - Dev-only safety check
- pkg/sync/mutex_dev.go - Dev-only lock timeout

**Framework Conventions (1★):**
- operator/*, config-controller/api - kubebuilder patterns

---

## Implementation Plan

### Step 1: Verification (30 min)
Run dependency analysis to determine actual busybox imports for:
- pkg/protoconv/resources/volumes
- pkg/gjson
- pkg/tlsprofile, pkg/httputil/proxy, etc.

### Step 2: Tier 2 Quick Wins (1-2 hours)
Implement guaranteed busybox candidates:
- clientconn useragent
- renderer asset
- dead code deletion
- postgres metrics

### Step 3: Tier 1 Conditional (1-2 hours)
If volume registry used by busybox: implement RegisterAll() pattern

### Step 4: Tier 3 Conditional (1-3 hours)
Implement verified utilities based on Step 1 analysis

---

## Success Metrics

**Phase 5 BusyBox Target:** 8-25 init() functions
**Effort:** 4-8 hours
**Separate scanner PR:** 5 init() functions, 2-4 hours

**Post-Phase 5 Status:**
- Migrated: 93-110 of 535 (17-21%) - busybox-focused
- Stubbed: 124 (23%) - compliance + GraphQL
- Scanner (separate): 5 additional
- Remaining: ~396-413 (74-77%)

**Key Wins:**
- ✅ Dead code removed
- ✅ Central-specific code properly isolated
- ✅ Volume registry cleaned up (if used)
- ✅ Unnecessary defaults removed

---

## Note for Scanner Team

Scanner binary has 5 easy-to-migrate init() functions (4/5 trivial):
1. scanner/cmd/scanner/main.go - Move to main()
2. scanner/enricher/nvd/nvd.go - URL parse → must()
3. scanner/enricher/csaf/.../zreader.go - Computed const
4. scanner/indexer/indexer.go - LRU cache → must()
5. scanner/matcher/matcher.go - Explicit registration

**Effort:** 2-4 hours
**Impact:** Scanner completely init()-free
**Recommendation:** Create separate PR/issue for scanner optimization
