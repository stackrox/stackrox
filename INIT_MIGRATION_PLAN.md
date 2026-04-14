# Init() Migration Plan - Step by Step

## Overview
Current state: 38 init() functions across 6 directories
Goal: Move all init() to explicit Init() functions called from component entry points

## Migration Strategy

### Phase 1: Config-Controller (2 init functions) - EASIEST START
**Why first**: Smallest, isolated component with clear entry point

Files:
1. `config-controller/app/app.go` - Move setup to explicit function
2. `config-controller/api/v1alpha1/policy_types.go` - Scheme registration

**Plan:**
- Create `config-controller/app/init.go` with `InitSchemes()`
- Move `SchemeBuilder.Register()` call from policy_types.go to init.go
- Call `InitSchemes()` from `config-controller/app/app.go:Run()`
- Remove `config-controller/` from gochecknoinits exclusion
- Verify with linter

**Estimated complexity**: LOW (2 files, clear pattern)

---

### Phase 2: Compliance (5 init functions)
**Why second**: Small scope, clear patterns (metrics + setup)

Files:
1. `compliance/virtualmachines/relay/metrics/metrics.go` - Metrics init
2. `compliance/collection/metrics/metrics.go` - Metrics init
3. `compliance/node/index/indexer.go` - Setup
4. `compliance/collection/file/user.go` - Setup
5. `compliance/collection/kubernetes/kubelet.go` - Setup

**Plan:**
- Create `compliance/cmd/compliance/app/init.go` with:
  - `initMetrics()` - consolidate both metrics files
  - `initCollectors()` - consolidate indexer, user, kubelet setup
- Call both from `compliance/cmd/compliance/app/app.go:Run()`
- Remove `compliance/` from gochecknoinits exclusion
- Verify

**Estimated complexity**: LOW-MEDIUM (5 files, 2 categories)

---

### Phase 3: Operator (4 init functions)
**Why third**: Kubernetes operator pattern, scheme registrations

Files:
1. `operator/internal/overlays/postrenderer_test.go` - Test (skip)
2. `operator/api/v1alpha1/central_types.go` - Scheme registration
3. `operator/api/v1alpha1/securedcluster_types.go` - Scheme registration
4. `operator/cmd/main.go` - Main setup

**Plan:**
- Create `operator/app/init.go` with:
  - `InitSchemes()` - consolidate both CRD scheme registrations
  - Move main.go init logic to `InitOperator()`
- Create `operator/app/app.go` if doesn't exist
- Call init functions from app.Run()
- Remove `operator/` from gochecknoinits exclusion (but keep test file exclusion)
- Verify

**Estimated complexity**: MEDIUM (operator-specific patterns, may need app structure)

---

### Phase 4: Central (11 init functions)
**Why fourth**: Most complex, many different patterns

Files:
1. `central/clusters/zip/render_test.go` - Test (skip)
2. `central/networkpolicies/graph/evaluator_test.go` - Test (skip)
3. `central/auth/internaltokens/service/metrics.go` - Metrics
4. `central/search/options/options.go` - Search options map
5. `central/alert/mappings/options.go` - Alert options map
6. `central/debug/service/service.go` - Debug service registration
7. `central/scannerdefinitions/handler/handler.go` - Handler registration
8. `central/detection/service/service_impl.go` - Service registration
9. `central/globaldb/v2backuprestore/formats/postgresv1/format.go` - Format registration
10. `central/cve/common/csv/handler.go` - CSV handler registration
11. `central/main.go` - Main setup

**Plan:**
- Extend `central/app/init.go` with:
  - `initServiceMetrics()` - auth/internaltokens metrics
  - `initSearchOptions()` - search options
  - `initAlertMappings()` - alert mappings
  - `initServiceRegistrations()` - debug, scanner, detection, backup format, CSV
  - Move main.go init logic to appropriate function
- Call all from `central/app/app.go:Run()`
- Remove `central/` from gochecknoinits exclusion (keep _test.go exclusion)
- Verify

**Estimated complexity**: HIGH (11 files, diverse patterns, already has app structure)

---

### Phase 5: Migrator (15 init functions) - SPECIAL CASE
**Why last**: Migration registration pattern is standardized

Files:
1. `migrator/migrations/m_*_to_m_*/migration.go` (14 files) - Migration registrations
2. `migrator/runner/runner.go` - Runner setup

**Pattern observed**: Each migration file has `init()` that registers itself

**Plan (two approaches):**

**Option A: Keep init() for migrations (recommended)**
- Migrations are auto-discovery pattern - init() is intentional
- Only migrate `migrator/runner/runner.go`
- Update exclusion to: `^(scanner/|generated/|migrator/migrations/)`
- Justification: 14 identical init() patterns, changing would require migration registry refactor

**Option B: Explicit registration**
- Create `migrator/migrations/all.go` with `RegisterAllMigrations()`
- Call each m_X_to_Y.Register() explicitly
- Refactor is large but more explicit
- Remove all migrator exclusions

**Recommendation**: Start with Option A (keep migration init() pattern), can refactor later if needed

**Estimated complexity**:
- Option A: LOW (1 file)
- Option B: HIGH (15 files, registry refactor)

---

### Phase 6: Image (1 init function) - TRIVIAL
Files:
1. `image/embed_charts_test.go` - Test file (already excluded by _test.go rule)

**Plan:**
- No action needed (test files already excluded)
- Remove `image/` from exclusion path

**Estimated complexity**: TRIVIAL

---

## Final gochecknoinits Exclusion Pattern

After all phases:
```yaml
- linters:
    - gochecknoinits
  # Allow init() only in scanner (separate binary), generated code, and migration auto-registration
  path: ^(scanner/|generated/|migrator/migrations/)
```

## Verification Process (After Each Phase)

1. Run golangci-lint on the directory:
   ```bash
   GOMEMLIMIT=10GiB .gotools/bin/golangci-lint run --verbose ./<directory>/...
   ```

2. Run unit tests for the directory:
   ```bash
   go test ./<directory>/...
   ```

3. Update `.golangci.yml` exclusion pattern to remove the directory

4. Run full golangci-lint to ensure no regressions:
   ```bash
   GOMEMLIMIT=10GiB .gotools/bin/golangci-lint run --verbose ./...
   ```

5. Commit with clear message about which directory was migrated

## Timeline Estimate

- Phase 1 (config-controller): 30 minutes
- Phase 2 (compliance): 1 hour
- Phase 3 (operator): 1.5 hours
- Phase 4 (central): 2 hours
- Phase 5 (migrator): 30 minutes (Option A) or 2 hours (Option B)
- Phase 6 (image): 5 minutes

**Total**: ~5-7 hours of focused work

## Rollout Strategy

1. Each phase is a separate commit
2. Each commit must pass CI (golangci-lint, unit tests)
3. Can be done over multiple PRs if needed:
   - PR 1: Phases 1-2 (config-controller + compliance)
   - PR 2: Phase 3 (operator)
   - PR 3: Phase 4 (central)
   - PR 4: Phases 5-6 (migrator + image)

## Next Step

Start with Phase 1: Config-Controller (easiest, validates the approach)
