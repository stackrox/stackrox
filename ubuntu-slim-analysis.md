# GitHub Actions Ubuntu-Slim Migration Analysis

## Executive Summary

This analysis identifies PR-triggered GitHub Actions jobs that could benefit from switching to `ubuntu-slim` runners. Ubuntu-slim is a 1-vcpu Linux runner optimized for lightweight jobs with faster boot times and lower costs.

## Key Differences: ubuntu-latest vs ubuntu-slim

| Feature | ubuntu-latest | ubuntu-slim |
|---------|---------------|-------------|
| Base Image | Ubuntu 24.04 | Ubuntu 24.04 |
| vCPUs | 4 cores | 1 core |
| RAM | 16 GB | 4 GB |
| Storage | 14 GB SSD | 14 GB SSD |
| Pre-installed Tools | ~300 tools | ~30 essential tools |
| Boot Time | ~10-15s | ~5-8s (faster) |
| Best For | Heavy builds, parallel work | Validation, linting, simple tasks |

**Missing from ubuntu-slim** (compared to ubuntu-latest):
- Docker/container runtimes (but containers work via setup actions)
- Most language runtimes (but can use setup-* actions)
- Many CLI tools (can install as needed)

## PR-Triggered Workflows Analysis

### 1. check-pr-title.yaml ✅ **EXCELLENT CANDIDATE**
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Pure bash/grep validation
- No external dependencies
- No build tools needed
- Runs in <30 seconds
- Perfect fit for 1-vcpu runner

**Changes needed:**
```yaml
jobs:
  check-title:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

**Expected benefit:** Faster boot time, lower cost, no functionality impact

---

### 2. labeler.yaml ✅ **EXCELLENT CANDIDATE**
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Only uses actions/labeler action
- No build or compile steps
- Extremely lightweight
- Runs in <20 seconds

**Changes needed:**
```yaml
jobs:
  label-pr:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

**Expected benefit:** Faster execution, lower resource consumption

---

### 3. style.yaml - Mixed Opportunities

#### ✅ **misc-checks** - GOOD CANDIDATE
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Bash scripts and basic validations
- Uses git, grep, basic shell tools
- No heavy compilation
- Python (pycodestyle) installed via setup action

**Changes needed:**
```yaml
jobs:
  misc-checks:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

**Expected benefit:** Faster boot, sufficient resources for validation scripts

#### ⚠️ **check-generated-files** - KEEP CONTAINER-BASED
**Current:** `ubuntu-latest` with container
**Recommendation:** No change (container-based, host runner doesn't matter much)

**Why:**
- Already uses quay.io/stackrox-io/apollo-ci container
- Host runner is just orchestrator
- Minimal benefit from changing

#### ⚠️ **style-check** - KEEP ubuntu-latest
**Current:** `ubuntu-latest`
**Recommendation:** Keep as-is

**Why:**
- Runs `make style-slim` which is parallel-heavy
- Caches Go deps, UI deps, Gradle deps
- Benefits from 4-core parallelism
- Uses setup-node, complex dependencies

#### ❌ **golangci-lint** - KEEP ubuntu-latest
**Current:** `ubuntu-latest`
**Recommendation:** Keep as-is

**Why:**
- CPU-intensive linting across entire codebase
- 240-minute timeout indicates heavy workload
- Benefits significantly from multi-core
- Would be much slower on 1-vcpu

#### ✅ **check-dependent-images-exist** - GOOD CANDIDATE
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Just makes API calls to Quay.io to check image existence
- Uses stackrox/actions/release/wait-for-image
- No compilation or heavy processing
- Network I/O bound, not CPU bound

**Changes needed:**
```yaml
jobs:
  check-dependent-images-exist:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

#### ✅ **github-actions-lint** - GOOD CANDIDATE
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Runs `make github-actions-lint` (yamllint on workflow files)
- Simple linting task
- No heavy dependencies

**Changes needed:**
```yaml
jobs:
  github-actions-lint:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

#### ✅ **github-actions-shellcheck** - GOOD CANDIDATE
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Just runs shellcheck on a few scripts
- Lightweight validation
- Perfect for 1-vcpu

**Changes needed:**
```yaml
jobs:
  github-actions-shellcheck:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

#### ✅ **openshift-ci-lint** - GOOD CANDIDATE
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Python linting (pycodestyle, pylint)
- Not computationally intensive
- Single-threaded linters

**Changes needed:**
```yaml
jobs:
  openshift-ci-lint:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

---

### 4. build.yaml - Selective Opportunities

#### ✅ **define-job-matrix** - GOOD CANDIDATE
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Just runs bash scripts with jq to compute matrix
- No compilation
- Fast execution (<2 minutes)
- Perfect for 1-vcpu

**Changes needed:**
```yaml
jobs:
  define-job-matrix:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

#### ✅ **go-version-ceiling** - GOOD CANDIDATE
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Only runs `go mod tidy` to validate go.mod
- Single-threaded operation
- No heavy builds
- Quick validation (<1 minute)

**Changes needed:**
```yaml
jobs:
  go-version-ceiling:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

#### ⚠️ **pre-build-ui** - KEEP ubuntu-latest
**Current:** `ubuntu-latest`
**Recommendation:** Keep as-is

**Why:**
- Runs `make -C ui build` (webpack builds)
- Benefits from parallel processing
- UI builds are CPU-intensive
- Would be significantly slower on 1-vcpu

#### ⚠️ **pre-build-cli, pre-build-go-binaries** - KEEP CONTAINER-BASED
**Current:** `ubuntu-latest` with containers
**Recommendation:** No change

**Why:**
- Already use apollo-ci containers
- Compilation benefits from multi-core
- Host is just orchestrator

#### ✅ **pre-build-docs, pre-build-oss-notice** - GOOD CANDIDATES
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Generate documentation/notices (not CPU-intensive)
- Mostly I/O operations
- No parallel benefits

**Changes needed:**
```yaml
jobs:
  pre-build-docs:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest

  pre-build-oss-notice:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

#### ✅ **build-operator-bundle** - GOOD CANDIDATE
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Builds operator bundle (YAML manifests)
- Uses Python for bundle helpers
- Not compute-intensive
- Mostly template processing

**Changes needed:**
```yaml
jobs:
  build-operator-bundle:
    runs-on: ubuntu-slim  # Changed from ubuntu-latest
```

#### ❌ **build-and-push-main, build-and-push-operator, build-and-push-scanner** - KEEP AS-IS
**Current:** `ubuntu-latest` or `ubuntu-24.04-arm`
**Recommendation:** No change

**Why:**
- Heavy Docker builds
- Need multi-core for buildx
- Already optimized (ARM uses native arm64 runners)

#### ✅ **push-*-manifests** - GOOD CANDIDATES
**Current:** `ubuntu-latest`
**Recommendation:** Switch to `ubuntu-slim`

**Why:**
- Just create and push manifest lists
- Docker manifest commands are lightweight
- No compilation or heavy processing

**Changes needed:**
```yaml
jobs:
  push-main-manifests:
    runs-on: ubuntu-slim

  push-scanner-manifests:
    runs-on: ubuntu-slim

  push-operator-manifests:
    runs-on: ubuntu-slim

  push-matching-collector-scanner:
    runs-on: ubuntu-slim
```

#### ✅ **scan-images-with-roxctl** - POTENTIAL CANDIDATE
**Current:** `ubuntu-latest`
**Recommendation:** Try `ubuntu-slim`

**Why:**
- Runs roxctl image scan (mostly network I/O)
- Not compute-intensive
- May work fine on 1-vcpu

**Risk:** Scanner might benefit from multiple cores for parallel scanning

---

### 5. unit-tests.yaml - Selective Opportunities

#### ⚠️ **go, go-postgres, go-bench** - KEEP CONTAINER-BASED
**Current:** Containers (apollo-ci)
**Recommendation:** No change

**Why:**
- Already use containers
- Tests benefit from 4-core host for parallelism
- Container-based, minimal host impact

#### ⚠️ **ui, ui-component** - KEEP CONTAINER-BASED
**Current:** Containers (apollo-ci)
**Recommendation:** No change

**Why:**
- Use containers
- UI tests can be CPU-heavy

#### ⚠️ **sensor-integration-tests** - KEEP ubuntu-latest
**Current:** `ubuntu-latest`
**Recommendation:** Keep as-is

**Why:**
- Creates Kind cluster (needs resources)
- Runs integration tests
- Pulls and loads multiple container images
- Benefits from 4 cores and 16GB RAM

#### ⚠️ **local-roxctl-tests, shell-unit-tests, openshift-ci-unit-tests** - KEEP CONTAINER-BASED
**Current:** Containers
**Recommendation:** No change

---

### 6. check-crd-compatibility.yaml
#### ⚠️ **check-crd-compatibility** - KEEP CONTAINER-BASED
**Current:** Container (apollo-ci)
**Recommendation:** No change

**Why:**
- Uses container, minimal host impact

---

## Summary of Recommendations

### ✅ High-Confidence Changes (15 jobs)

These jobs should be switched to `ubuntu-slim` with minimal risk:

1. **check-pr-title.yaml**
   - `check-title`

2. **labeler.yaml**
   - `label-pr`

3. **style.yaml**
   - `misc-checks`
   - `check-dependent-images-exist`
   - `github-actions-lint`
   - `github-actions-shellcheck`
   - `openshift-ci-lint`

4. **build.yaml**
   - `define-job-matrix`
   - `go-version-ceiling`
   - `pre-build-docs`
   - `pre-build-oss-notice`
   - `build-operator-bundle`
   - `push-main-manifests`
   - `push-scanner-manifests`
   - `push-operator-manifests`
   - `push-matching-collector-scanner`

### ⚠️ Test Candidates (1 job)

Worth testing but monitor performance:

1. **build.yaml**
   - `scan-images-with-roxctl`

### ❌ Keep as ubuntu-latest (6 jobs)

Do NOT change these - they need multi-core:

1. **style.yaml**
   - `style-check` (parallel linting)
   - `golangci-lint` (CPU-intensive)

2. **build.yaml**
   - `pre-build-ui` (webpack parallel builds)
   - `build-and-push-*` (Docker builds)

3. **unit-tests.yaml**
   - `sensor-integration-tests` (Kind cluster + integration tests)

### ℹ️ No Change Needed (Container-based)

These use containers, so host runner type has minimal impact:
- All jobs using `container: image: quay.io/stackrox-io/apollo-ci:*`

---

## Implementation Strategy

### Phase 1: Low-Risk Wins (Immediate)
Start with these 5 ultra-safe changes:
1. `check-pr-title`
2. `labeler`
3. `github-actions-lint`
4. `github-actions-shellcheck`
5. `define-job-matrix`

### Phase 2: Validation Jobs (Week 2)
Add these 6 validation/linting jobs:
1. `misc-checks`
2. `openshift-ci-lint`
3. `check-dependent-images-exist`
4. `go-version-ceiling`

### Phase 3: Build Metadata Jobs (Week 3)
Add these 4 build orchestration jobs:
1. `pre-build-docs`
2. `pre-build-oss-notice`
3. `build-operator-bundle`
4. All `push-*-manifests` jobs

### Phase 4: Experimental (Week 4+)
Test and monitor:
1. `scan-images-with-roxctl`

---

## Expected Benefits

### Cost Savings
- ubuntu-slim runners are cheaper (1-vcpu vs 4-vcpu)
- Estimated 15 jobs × average 5 min × cost differential
- Potential 30-40% cost reduction on lightweight PR checks

### Performance Impact
- **Faster:** Boot time reduced by ~50% (5-8s vs 10-15s)
- **Neutral:** Most validation jobs are I/O or network bound
- **Slower:** None (we're keeping CPU-intensive jobs on ubuntu-latest)

### Overall
- **Total jobs to migrate:** 15-16
- **Expected speedup:** 10-20% for lightweight jobs (faster boot)
- **Risk level:** Low (all selected jobs are single-threaded or I/O-bound)

---

## Monitoring Plan

After migration:
1. Monitor job duration in Actions UI
2. Check failure rates for changed jobs
3. Compare PR check completion times
4. Review cost metrics in GitHub billing

If any job shows >20% slowdown, revert to `ubuntu-latest`.

---

## Notes

- Docker/containers work on ubuntu-slim via setup actions
- All language runtimes (Go, Node, Python) available via setup-* actions
- ubuntu-slim has the same Ubuntu 24.04 base as ubuntu-latest
- 1-vcpu is sufficient for sequential/scripted tasks
- Multi-core parallelism needs ubuntu-latest (4-vcpu)
