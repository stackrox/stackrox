# Native CI Selector - Prototype

**Status:** Proof of Concept  
**Date:** 2026-10-26  
**Branch:** `piotr/ROX-36969-ci-prototype-2` (co-located with resolver/dispatcher prototype)

## Overview

This is an alternative approach to Piotr's resolver/dispatcher architecture (PR #23166). Instead of custom Python resolver + dispatchers, this uses **native platform features** with **generated implementations** from a single source of truth.

### Key Differences from Resolver/Dispatcher

| Aspect | Resolver/Dispatcher | Native Approach (This) |
|--------|-------------------|----------------------|
| Architecture | Custom Python resolver + 2 dispatchers | Config generator + native conditions |
| Config | `ci/test-domains.toml` (TOML) | `ci/test-selection-rules.yaml` (YAML) |
| GHA Integration | Custom dispatcher reads decision file | Native job `if:` conditions (generated) |
| Prow Integration | Custom dispatcher reads decision file | Bash functions (generated) |
| Code Generation | None (runtime evaluation) | On-demand (cheap, 1-2 seconds) |
| Generated Artifacts | Decision file + logs (runtime) | Workflow YAML + bash script (repo) |
| Single Source of Truth | ✅ TOML config | ✅ YAML config |
| `/test` override | ✅ Supported | ✅ Supported |
| Label support | ✅ Full | ✅ Full |

## How It Works

1. **Single config:** `ci/test-selection-rules.yaml` defines all rules
2. **Generator tool:** `ci/select/generator/` (Go) reads config and generates:
   - `.github/workflows/detect-changes-generated.yml` (enhanced)
   - `ci/prow-native-gate.sh` (bash functions)
3. **Generation:** On-demand via `make generate-ci-selection` (1-2 seconds)
4. **Integration:**
   - GitHub Actions: Jobs use `if:` conditions with domain flags from detect-changes
   - Prow: `dispatch.sh` sources generated bash script, calls `ci_should_run_job <job>`

## Files in This Prototype

```
ci/
├── test-selection-rules.yaml          # Single source of truth (YAML config)
├── prow-native-gate.sh                # GENERATED - Prow bash functions
└── select/
    └── generator/                     # Go tool that generates from config
        ├── main.go
        ├── types.go
        ├── github_actions.go
        ├── prow.go
        ├── go.mod
        └── generator                  # Compiled binary

.github/workflows/
└── detect-changes-generated.yml       # GENERATED - Enhanced detect-changes workflow

Makefile                               # Added: `make generate-ci-selection` target
```

## Usage

### Editing Rules

Edit `ci/test-selection-rules.yaml`:

```yaml
rules:
  - name: sensor-subsystem
    description: Sensor code changes need sensor tests
    when:
      any_file_matches:
        - '^sensor/'
        - '^pkg/sensor/'
    action:
      run:
        - sensor-integration-tests
        - go
      skip:
        - go-postgres
```

### Regenerating Outputs

```bash
make generate-ci-selection
```

This rebuilds the generator and regenerates both files. **Commit the generated files** so they're visible in PRs.

### CI Validation

Add a CI job to ensure generated files are up-to-date:

```yaml
- name: Validate CI selection rules
  run: |
    make generate-ci-selection
    if ! git diff --exit-code; then
      echo "Error: Generated CI selection files out of date"
      echo "Run: make generate-ci-selection"
      exit 1
    fi
```

## Config Format

See `ci/test-selection-rules.yaml` for full example. Key sections:

### Jobs List

```yaml
jobs:
  - style-check
  - go
  - sensor-integration-tests
  # ... all known jobs
```

### Defaults

```yaml
defaults:
  style-check: run  # Runs on every PR by default
  gke-qa-e2e-tests: skip  # Only runs on master/releases
```

### Rules

```yaml
rules:
  - name: rule-name
    description: What this rule does
    when:
      label: ci-run-all-tests            # Label-based
      # OR
      any_file_matches:                  # Path-based
        - '^sensor/'
      # OR
      every_file_matches:                # All files must match
        - '\.md$'
      # OR
      any_file_changed: true             # Any change
    action:
      run:
        - job1
        - job2
        - all  # Special value for all jobs
      skip:
        - job3
      except:
        - job4  # Exception to skip (job4 runs even if "all" skipped)
```

## GitHub Actions Integration

**Workflows use generated domain flags:**

```yaml
jobs:
  detect-changes:
    uses: ./.github/workflows/detect-changes-generated.yml

  sensor-tests:
    needs: detect-changes
    if: |
      github.event_name != 'pull_request' ||
      needs.detect-changes.outputs.sensor_subsystem == 'true' ||
      needs.detect-changes.outputs.area_sensor_label == 'true' ||
      contains(github.event.pull_request.labels.*.name, 'ci-run-all-tests')
    runs-on: ubuntu-24.04
    steps: ...
```

Domain flags are automatically generated from rules in config.

## Prow Integration

**dispatch.sh sources generated script:**

```bash
#!/usr/bin/env bash

ci_job="$1"
shift

# NEW: Check if job should run based on generated rules
source "$ROOT/ci/prow-native-gate.sh"
if ! ci_should_run_job "$ci_job"; then
    info "Skipping ${ci_job} based on test selection rules"
    exit 0
fi

# Rest of existing dispatch.sh logic
openshift_ci_mods
openshift_ci_import_creds
# ...
```

The generated `ci_should_run_job` function:
- ✅ Checks `/test` overrides (highest priority)
- ✅ Evaluates all rules from config
- ✅ Applies clash resolution (run wins)
- ✅ Falls back to defaults
- ✅ Fails open on errors

## Testing Locally

### Test GitHub Actions Generation

```bash
make generate-ci-selection
cat .github/workflows/detect-changes-generated.yml
```

### Test Prow Gate Script

```bash
# Simulate Prow environment
export PULL_BASE_SHA="origin/master"
export PULL_PULL_SHA="HEAD"
export PULL_NUMBER="12345"
export REPO_OWNER="stackrox"
export REPO_NAME="stackrox"

source ci/prow-native-gate.sh

if ci_should_run_job "sensor-integration-tests"; then
    echo "Job should run"
else
    echo "Job should skip"
fi
```

## Design Decisions

### 1. Generate on-demand (not checked-in)
- **Decision:** Check in generated files (committed to repo)
- **Why:** Visible in PRs, easy to review what changed, CI validates up-to-date
- **Tradeoff:** Must remember to regenerate (CI enforces this)

### 2. Config format: YAML
- **Decision:** YAML over TOML
- **Why:** Team familiarity (already use YAML everywhere)

### 3. Generator language: Go
- **Decision:** Go over Python
- **Why:** Team's primary language, type safety, fast compilation

### 4. Clash resolution: Run wins
- **Decision:** If run + skip conflict → run the job
- **Why:** False positives better than false negatives (tests validate correctness)

### 5. Generation cost
- **Measured:** ~1-2 seconds to regenerate both files
- **Conclusion:** Cheap enough to do on-demand in CI

## Comparison with Resolver/Dispatcher

### Advantages of Native Approach

✅ **Simpler mental model:** No decision file contract, no resolver algorithm to learn  
✅ **Transparent:** Generated code visible in repo, not opaque Python execution  
✅ **Native GHA:** Uses standard `if:` conditions, familiar to team  
✅ **Less runtime overhead:** No Python execution in every workflow  
✅ **Type safety:** Go generator vs. Python runtime errors

### Advantages of Resolver/Dispatcher

✅ **More flexible:** Arbitrary Python logic, easier to express complex rules  
✅ **Runtime evaluation:** No need to regenerate, rules applied dynamically  
✅ **Unified implementation:** Same resolver for both GHA and Prow  
✅ **Battle-tested:** Piotr's prototype is more complete

### Trade-offs

- **Complexity:** Native approach trades "runtime complexity" for "generation complexity"
- **Maintainability:** Generated bash is verbose but reviewable; Python is concise but opaque
- **Iteration speed:** Both require config change + testing; native requires regeneration step

## Next Steps

### To Complete Prototype

1. ✅ **Config schema:** `ci/test-selection-rules.yaml` created
2. ✅ **Generator tool:** Go tool implemented and working
3. ✅ **Sample rules:** Ported from Piotr's `test-domains.toml`
4. ⬜ **Integrate with workflows:** Modify `style.yaml`, `unit-tests.yaml`, `e2e-dispatch.yaml`
5. ⬜ **Test on PRs:** Compare behavior with resolver/dispatcher
6. ⬜ **Documentation:** How to add rules, troubleshooting

### Evaluation Criteria

Test both approaches on real PRs and compare:

1. **Understandability:** Which is easier for engineers to learn?
2. **Maintainability:** Which is easier to debug and modify?
3. **Correctness:** Do they produce same results?
4. **Performance:** Any noticeable difference in CI time?
5. **Team preference:** Which does the team prefer?

## Open Questions

1. **Validation:** Should generator validate regex patterns compile?
2. **Sub-suite selection:** Extend to support test tags (e.g., `BAT_POLICY`)?
3. **Error handling:** How to report invalid configs clearly?
4. **Migration:** Can we share rules between approaches during evaluation?

## References

- Original prototype (resolver/dispatcher): https://github.com/stackrox/stackrox/pull/23166
- Design discussion: `/tmp/native-ci-selector-plan.md` (full design doc)
- Epic folder: `~/epic/ROX-36969/`
