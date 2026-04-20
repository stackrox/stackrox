# roxctl Central Generate Migration Analysis Plan

## Goal
Understand which `roxctl central generate` options affect manifest output, so users migrating to the operator can specify equivalent settings in their Central CR.

## Working Directory
All work will be done in `./MIGRATION/`

## Modes to Analyze
1. `roxctl central generate openshift pvc`
2. `roxctl central generate k8s pvc`
3. `roxctl central generate openshift hostpath`
4. `roxctl central generate k8s hostpath`

---

## Phase 1: Discover Available Options

### 1.1 Capture Help Output
For each of the 4 modes:
- Run: `roxctl central generate <platform> <storage> --help > help-<platform>-<storage>.txt`
- Files to create:
  - `help-openshift-pvc.txt`
  - `help-k8s-pvc.txt`
  - `help-openshift-hostpath.txt`
  - `help-k8s-hostpath.txt`

### 1.2 Compare Help Outputs
- Run `diff -u` between each pair of help files
- Understand which options are mode-specific vs. common across all modes
- Document findings

### 1.3 Create Master List
Create `MASTER_OPTIONS_LIST.md` with structure:
```markdown
## Option: --option-name

**Available in:** openshift-pvc, k8s-pvc, openshift-hostpath, k8s-hostpath
**Description:** (from help text)
**Impact:** (to be filled in Phase 4)
**Detection method:** (to be filled in Phase 4)
```

---

## Phase 2: Establish Baselines

### 2.1 Generate Baseline Manifests
For each mode:
- Run: `roxctl central generate <platform> <storage> --output-dir baseline-<platform>-<storage>`
- Directory structure:
  ```
  MIGRATION/
    baselines/
      openshift-pvc/
      k8s-pvc/
      openshift-hostpath/
      k8s-hostpath/
  ```

### 2.2 Inspect Baseline Contents
- Document what files/resources are generated in each mode
- Note any differences in baseline outputs between modes

---

## Phase 3: Identify Random/Non-Deterministic Values

### 3.1 Generate Duplicate Baselines
For each mode:
- Run the same command twice with no options
- Save to `baseline2-<platform>-<storage>/`
- Compare: `diff -ru baselines/<mode>/ baseline2-<mode>/`

### 3.2 Document Random Elements
Add to `MASTER_OPTIONS_LIST.md`:
- List of values that change between runs (likely: secrets, tokens, certificates)
- Patterns to ignore during option impact analysis

---

## Phase 4: Analyze Each Option's Impact

For **each option** in the Master List:

### 4.1 Generate Test Manifests
For each mode where the option is available:
- Determine appropriate test value for the option
- Run: `roxctl central generate <platform> <storage> --<option>=<value> --output-dir test-<option>-<platform>-<storage>`

### 4.2 Compare Against Baseline
- Run: `diff -ru baselines/<mode>/ test-<option>-<mode>/ > diffs/<option>-<mode>.diff`
- Save diff output for analysis

### 4.3 Analyze Impact
For each diff:
- Ignore random values identified in Phase 3
- Identify what resources/fields changed
- Classify impact:
  - **No impact** - option doesn't affect manifests
  - **Resource modification** - changes existing resource fields
  - **Resource addition/removal** - adds/removes entire resources
  - **Configuration change** - affects ConfigMap, environment variables, etc.

### 4.4 Design Detection Method
Create a `kubectl get` or `kubectl get <resource> -o jsonpath=...` command that:
- Can be run against a live cluster
- Produces minimal, human-readable output
- Clearly indicates whether the option was used and with what value

### 4.5 Update Master List
For each option, add:
- **Impact:** Concise description of changes
- **Detection method:** The kubectl command
- **Example:** Sample output showing option enabled vs. disabled

---

## Phase 5: Final Deliverables

### 5.1 Master Options List
Complete `MASTER_OPTIONS_LIST.md` with all findings

### 5.2 Summary Report
Create `SUMMARY.md` containing:
- Options that affect manifests (grouped by impact type)
- Options that have no impact (can be ignored for migration)
- Mode-specific options
- Recommendations for migration tool/guide

### 5.3 Cleanup
Organize working files:
```
MIGRATION/
  PLAN.md (this file)
  MASTER_OPTIONS_LIST.md (main deliverable)
  SUMMARY.md (summary report)
  help-outputs/ (help text files)
  baselines/ (baseline manifests)
  test-outputs/ (test manifests for each option)
  diffs/ (diff outputs)
```

---

## Execution Strategy

### Parallelization
- Phase 1-3: Sequential (build foundation)
- Phase 4: Can test options in parallel, but analyze sequentially to maintain focus

### Error Handling
- If `roxctl` fails for an option, document the error in Master List
- If diff is too large, create separate detailed analysis file

### Quality Checks
- Verify each option is tested in all applicable modes
- Cross-reference Master List against help output for completeness
- Test detection methods against actual baseline manifests

---

## Questions / Clarifications Needed

1. Should I use a specific version of roxctl, or the one currently in PATH?
2. For options requiring values (like `--external-endpoint`), should I use specific test values or document what values to test?
3. Should I focus only on options that likely affect manifests, or exhaustively test every single flag (including things like `--verbose`)?
4. Are there specific options you already know are important that I should prioritize?

---

## Estimated Artifacts

- ~4 help text files
- ~4 baseline directories with manifests
- ~N test output directories (where N = number of options × modes where applicable)
- ~N diff files
- 1 comprehensive Master Options List
- 1 summary report
