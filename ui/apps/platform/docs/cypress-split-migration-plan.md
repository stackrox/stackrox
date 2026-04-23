# Migration Plan: cypress-parallel → cypress-split

## Executive Summary

**Current Issue**: cypress-parallel v0.15.0 has a race condition causing random test distribution failures
**Short-term Fix**: `--strictMode false` (commit e256edf2d3)
**Permanent Solution**: Migrate to cypress-split (designed for CI/CD, no race conditions)

## Why cypress-split?

| Feature | cypress-parallel | cypress-split |
|---------|-----------------|---------------|
| **Use Case** | Single machine, multiple processes | CI/CD, multiple jobs/machines |
| **Reliability** | Race conditions in file distribution | Deterministic distribution |
| **CI/CD Support** | Documentation says "cannot be used with CI/CD" | Designed for CI/CD |
| **Maintenance** | Community plugin, 67 open issues | Official Cypress ecosystem |
| **Load Balancing** | Via weights file (unreliable) | Timing-based (optional, reliable) |

## Implementation Plan

### Phase 1: Local Setup & Testing (1-2 hours)

1. **Install cypress-split**
   ```bash
   cd ui/apps/platform
   npm install --save-dev cypress-split
   ```

2. **Configure in cypress.config.js**
   ```javascript
   const cypressSplit = require('cypress-split')

   module.exports = defineConfig({
     e2e: {
       setupNodeEvents(on, config) {
         cypressSplit(on, config)
         return config  // MUST return config!
       },
     },
   })
   ```

3. **Update package.json**
   ```json
   {
     "scripts": {
       "test-e2e-split": "cypress run --reporter mocha-multi-reporters --reporter-options configFile=cypress/mocha.config.js"
     }
   }
   ```

4. **Test locally with Docker**
   ```bash
   # Simulate 4 parallel jobs
   SPLIT=4 SPLIT_INDEX=0 npm run test-e2e-split &
   SPLIT=4 SPLIT_INDEX=1 npm run test-e2e-split &
   SPLIT=4 SPLIT_INDEX=2 npm run test-e2e-split &
   SPLIT=4 SPLIT_INDEX=3 npm run test-e2e-split
   ```

### Phase 2: CI/CD Integration (2-3 hours)

**Challenge**: StackRox uses Prow/OpenShift CI, not GitHub Actions

**Investigation needed**:
1. Where is the 4-thread parallelization currently configured?
   - Check `.prow.yaml` or similar CI config
   - Find how `npm run test-e2e-parallel` is invoked

2. How to pass `SPLIT` and `SPLIT_INDEX` to each job?
   - Prow parallel jobs syntax
   - Environment variable injection

**Likely pattern** (needs verification):
```yaml
# In Prow config
- name: ui-e2e-tests
  parallelism: 4
  env:
    - name: SPLIT
      value: "4"
    - name: SPLIT_INDEX
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['prow.k8s.io/job-index']
```

### Phase 3: Optimization (Optional, 1 hour)

**Add timing-based load balancing**:

1. First run generates `timings.json`:
   ```bash
   SPLIT_OUTPUT_FILE=cypress-timings.json npm run test-e2e-split
   ```

2. Commit `cypress-timings.json` to repo

3. Future runs use it automatically:
   ```bash
   SPLIT_FILE=cypress-timings.json npm run test-e2e-split
   ```

4. Benefits:
   - Equalizes job duration (e.g., job1: 5min, job2: 5min vs job1: 8min, job2: 2min)
   - Faster overall CI time

### Phase 4: Cleanup (30 min)

1. Remove cypress-parallel:
   ```bash
   npm uninstall cypress-parallel
   ```

2. Delete generated files:
   ```bash
   rm -rf runner-results/
   rm multi-reporter-config.json
   ```

3. Update documentation

## Rollout Strategy

### Option A: Big Bang (Recommended)
- Implement all phases in one PR
- Test locally first
- Deploy to CI once verified
- **Risk**: Medium (new tool)
- **Benefit**: Clean cutover, no technical debt

### Option B: Gradual
- Keep both systems running
- Run cypress-split in parallel with cypress-parallel for 1 week
- Compare results
- Switch over once confident
- **Risk**: Low
- **Benefit**: Safer, but more complex CI config

## Success Criteria

- ✅ All 122 tests run on every CI run
- ✅ No "Some test suites likely terminated" errors
- ✅ Deterministic distribution (same tests always on same job index)
- ✅ CI time ≤ current time (ideally faster with load balancing)
- ✅ Zero flakiness due to test distribution

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Prow doesn't support required env vars | High | Research Prow docs, ask in #prow Slack |
| cypress-split incompatible with mocha-multi-reporters | Medium | Test locally first, check GitHub issues |
| Timing file causes merge conflicts | Low | Add to .gitignore, regenerate in CI |
| Slower than cypress-parallel | Low | Benchmark first, add timing file |

## Next Steps

1. **Immediate**: Monitor PR #20165 CI with `--strictMode false` fix
2. **This week**: Phase 1 - Local testing with cypress-split
3. **Next week**: Phase 2 - CI integration (needs Prow config investigation)
4. **Week after**: Deploy to production

## References

- [cypress-split GitHub](https://github.com/bahmutov/cypress-split)
- [Free parallelization blog](https://glebbahmutov.com/blog/cypress-parallel-free/)
- [cypress-parallel issue #90](https://github.com/tnicola/cypress-parallel/issues/90)
- [Comparison article](https://laerteneto.medium.com/cypress-parallelization-tools-and-approaches-from-a-high-perspective-1599cc168ad2)
