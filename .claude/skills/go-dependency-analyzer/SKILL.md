---
name: go-dependency-analyzer
description: Analyzes Go dependencies to determine usage in production code, what functionality is used, and where it's located. Use when user asks "verify where we use [dependency]", "is [dependency] used in production", "analyze dependency [name]", "what uses [package]", "dependency analysis", mentions CVE numbers, security vulnerabilities, or needs to understand dependency impact for triage, upgrades, or removal decisions.
---

# Go Dependency Analyzer

Analyzes Go dependencies to determine production impact, used functionality, and code locations. Supports CVE triage, security issue assignment, dependency upgrade decisions, and understanding what would break if a dependency is removed or changed.

## Key Features

- **Branching Workflow:** Automatically detects direct vs transitive dependencies and uses appropriate analysis path
- **Wrapper Pattern Detection:** Identifies when dependencies are wrapped by internal packages (e.g., zap via pkg/logging)
- **goda Visualization:** Mandatory dependency tree visualization for understanding import chains
- **Transitive Chain Analysis:** Shows full dependency path: Our code → Intermediate → Dependency
- **Replace Directive Tracking:** Detects StackRox forks and version gaps
- **go mod why Integration:** Includes dependency justification in all reports

## Instructions

### Step 1: Identify the Dependency

When given a dependency name, CVE, or package:
1. Extract the Go module path (e.g., `github.com/jackc/pgx/v5`)
2. If given a CVE number, search for the affected package name first
3. Normalize the module path (handle version suffixes like `/v2`, `/v5`)
4. Accept partial names (e.g., "pgx" → find full path `github.com/jackc/pgx/v5`)

### Step 2: Check Dependency Presence & Type

IMPORTANT: Work in current directory - do not change directories (respects git worktrees).

Run these commands in parallel:

```bash
# Check if dependency is in go.mod (direct or indirect)
grep -i "module-name" go.mod

# Check for replace directives (StackRox forks)
grep -i "module-name" go.mod | grep "=>"

# Check if indirect
grep -i "module-name" go.mod | grep "// indirect"
```

**If NOT found:** Respond immediately that the dependency is not used - issue can be closed.

**If replaced:** Note the fork/replacement in analysis (e.g., `go.uber.org/zap => github.com/stackrox/zap`).

**CRITICAL DECISION POINT:**

Check if dependency is marked `// indirect`:
- **If YES:** Follow **Step 2A: Transitive Dependency Analysis Path**
- **If NO:** Follow **Step 3: Direct Dependency Analysis Path**

### Step 2A: Transitive Dependency Analysis Path

**ONLY use this path for indirect dependencies.**

```bash
# Show why this transitive dependency is needed
go mod why -m module-name

# Show full dependency chain using go mod graph
go mod graph | grep "module-name"

# Visualize what pulls this dependency
goda graph "reach(./..., module-name/...)" | head -30
```

**Analysis for transitive dependencies:**

1. **Identify the chain:** Who pulls this dependency?
   - Format: `Our package → Intermediate package → module-name`
   - Example: `central/externalbackups → cloud.google.com/go/storage → protoc-gen-validate`

2. **Check if we use the intermediate package:**
   ```bash
   grep -r "intermediate-package" --include="*.go" --exclude-dir=vendor --exclude="*_test.go" | wc -l
   ```

3. **Report format for transitive:**
   ```
   ## Dependency Analysis: [Module Name] (TRANSITIVE)

   **Status:** TRANSITIVE ONLY - Not directly used

   **Dependency Chain:**
   ```
   Our code
       ↓ imports
   [intermediate package]
       ↓ depends on
   [module-name] v[version]
   ```

   **Why it exists:**
   - [Explain what intermediate package needs it for]
   - Used by: [List our files using intermediate package]

   **Action:**
   - No direct maintenance needed
   - Updated automatically when [intermediate package] updates
   - Monitor for security issues but low priority (transitive)
   ```

**SKIP to Step 7 (Generate Report)** after transitive analysis.

### Step 3: Direct Dependency Analysis Path

**ONLY use this path for direct dependencies (not marked `// indirect`).**

```bash
# Confirm it's direct
go list -m module-name

# MANDATORY: Show why dependency is needed
go mod why -m module-name

# Visualize dependency graph with goda
goda graph "reach(./..., module-name/...)" | head -30

# Check how many packages use it
goda list "reach(./..., module-name/...)" | cut -d: -f1 | sort -u | wc -l
```

**Analysis:**
- **Direct dependency**: We explicitly import it
- **go mod why output**: MANDATORY - include in final report
- **goda visualization**: Shows which of our packages import it

### Step 4: Find Actual Usage Locations (Direct Dependencies Only)

**First: Check for wrapper packages**

Many dependencies are wrapped by internal `pkg/` packages. Check if this is the case:

```bash
# Search for wrapper in pkg/
grep -r '"module-name' pkg/ --include="*.go" --exclude-dir=vendor | head -10

# If found wrapper, also count wrapper usage
grep -r 'pkg/wrapper-name"' --include="*.go" --exclude-dir=vendor --exclude="*_test.go" | wc -l
```

**Example:** `go.uber.org/zap` is wrapped by `pkg/logging` - need to count BOTH:
- Direct imports of zap (usually just in pkg/logging)
- Imports of pkg/logging (actual usage across codebase)

**Then: Find direct imports**

```bash
# Find all direct imports of the package
grep -r '"module-name' --include="*.go" --exclude-dir=vendor | cut -d: -f1 | sort -u | head -20

# Separate production vs test usage
grep -r '"module-name' --include="*.go" --exclude="*_test.go" --exclude-dir=vendor | cut -d: -f1 | sort -u | wc -l
grep -r '"module-name' --include="*_test.go" --exclude-dir=vendor | cut -d: -f1 | sort -u | wc -l

# Use goda to show files that import it
goda list "reach(./..., module-name/...)" | head -30
```

**If wrapper detected:**

Also analyze wrapper usage:

```bash
# Count wrapper package users
grep -r 'pkg/wrapper"' --include="*.go" --exclude-dir=vendor --exclude="*_test.go" | wc -l

# Find components using wrapper
grep -r 'pkg/wrapper"' --include="*.go" --exclude-dir=vendor --exclude="*_test.go" | cut -d: -f1 | cut -d/ -f1 | sort -u
```

### Step 5: Analyze Used Functionality (Direct Dependencies Only)

**Optional: GitNexus Integration**

If GitNexus MCP is available in the repository, PreToolUse:Bash hooks will automatically provide context like:
```
[GitNexus] 2 related symbols found:
zapLogConverter (pkg/logging/log_converter_impl.go)
```

Use this context to identify key files and functions quickly. This is supplementary - proceed with manual analysis regardless.

**Manual analysis:**

Use gopls MCP tools for precise analysis when available:

```bash
# Use mcp__gopls__go_search to find symbol usage
# Search for types, functions from the vulnerable package
```

For each file that imports the dependency:
1. Use `mcp__gopls__go_file_context` to see what's imported
2. Use `mcp__gopls__go_symbol_references` to find specific function calls
3. Use `Grep` with function names from CVE description
4. Reference GitNexus hook context if it appears

**Example for CVE analysis:**
```bash
# If CVE mentions "QueryRow" vulnerability in pgx
grep -r "QueryRow" --include="*.go" --exclude-dir=vendor | head -20

# If CVE mentions specific prometheus metric types
grep -r "NewCounter\|NewGauge\|NewHistogram" --include="*.go" --exclude-dir=vendor | head -10
```

### Step 6: Determine Production Impact

**Production code if:**
- Used in `*.go` files (not `*_test.go`)
- Not in `/test/`, `/qa-tests-backend/`, `/tools/` directories
- Imported by main application packages (`/central/`, `/sensor/`, `/scanner/`, `/roxctl/`)

**Non-production if:**
- Only in `*_test.go` files
- Only in `/qa-tests-backend/`, `/tools/`, `/scripts/`
- Only transitive through test dependencies

### Step 7: Generate Analysis Report

**For DIRECT dependencies:**

```
## Dependency Analysis: [Dependency Name or CVE-ID]

**Dependency:** module-path v[version]
**Status:** USED IN PRODUCTION | USED IN TESTS ONLY | WRAPPER PATTERN

**Usage Summary:**
- Direct dependency: YES
- Production code files: [count] files (direct imports)
- Wrapper pattern: [if applicable: "+ [count] files via pkg/wrapper"]
- Test code files: [count] files
- Primary components: [list: central, sensor, scanner, roxctl, etc.]

**Why Needed (go mod why):**
```
[MANDATORY: Include go mod why -m output here]
github.com/stackrox/rox/path/to/package
    imports github.com/example/dependency
```

**Specific Functionality Used:**
- [List actual functions/types imported and used]
- CVE affects: [specific function from CVE if applicable]
- We use: [functions we actually call]

**Replace Directive:**
- Original: [module path if replaced]
- Fork/Replace: [replacement path from go.mod]
- Version Gap: [if fork behind: "Fork at v1.18, upstream at v1.27"]
- Reason: [if known, note why fork exists - check commit history]

**Locations:**
[List key files with line numbers where possible]
- central/path/file.go:123 - uses QueryRow()
- sensor/pkg/file.go:456 - uses Connect()

[If wrapper pattern detected:]
**Wrapper Usage:**
- Wrapped by: pkg/wrapper-name
- Components using wrapper: [list]
- Total indirect users: [count] files

**Recommended Action:**
- [ ] Assign to [team] - production impact in [component]
- [ ] Low priority - test code only
- [ ] Close - dependency not used
- [ ] Close - vulnerable functionality not used
- [ ] Review fork - sync with upstream [if fork detected]

**Team Assignment:**
Based on component usage: [suggest team from component ownership]
```

**For TRANSITIVE dependencies (from Step 2A):**

```
## Dependency Analysis: [Dependency Name] (TRANSITIVE)

**Dependency:** module-path v[version]
**Status:** TRANSITIVE ONLY - Not directly imported

**Dependency Chain:**
```
Our code (StackRox)
    ↓ imports
[intermediate-package] v[version]
    ↓ depends on
[module-name] v[version]
```

**Why It Exists:**
[From go mod why output - explain the chain]

**Our Usage of Intermediate Package:**
- We import: [intermediate-package]
- Used in: [count] production files
- Components: [list components using intermediate]
- Files: [list key files]

**What Intermediate Uses It For:**
- [Brief explanation - e.g., "GCS SDK uses it for proto validation"]
- [CVE impact: "If CVE affects this, check if intermediate is vulnerable"]

**Recommended Action:**
- [x] No direct action needed - transitive dependency
- [ ] Monitor for security updates (managed by [intermediate-package])
- [ ] Low priority - updated automatically with [intermediate-package]

**Team Assignment:**
N/A - Managed via [intermediate-package] updates
[If CVE: assign to team owning intermediate package usage]
```

## Advanced Techniques

### Analyzing multiple dependencies efficiently

When asked to analyze multiple dependencies (e.g., "check pgx, docker, and zap"):

**Step 1: Batch the go.mod checks**
```bash
# Check all at once
grep -E "pgx|docker|zap" go.mod

# Check for replaces
grep -E "pgx|docker|zap" go.mod | grep "=>"
```

**Step 2: Count files in parallel**
```bash
# Production files for each
grep -r "jackc/pgx" --include="*.go" --exclude="*_test.go" --exclude-dir=vendor | cut -d: -f1 | sort -u | wc -l
grep -r '"github.com/docker' --include="*.go" --exclude="*_test.go" --exclude-dir=vendor | cut -d: -f1 | sort -u | wc -l
grep -r '"go.uber.org/zap"' --include="*.go" --exclude="*_test.go" --exclude-dir=vendor | cut -d: -f1 | sort -u | wc -l
```

**Step 3: Present as summary table**
Create a comparison table showing all dependencies side-by-side for easy decision-making.

### Use goda for dependency trees

IMPORTANT: goda commands work in current directory automatically.

```bash
# Show what imports this dependency
goda graph "reach(./..., module-name/...)" | head -20

# Show transitive dependency path
goda graph "module-name/...:deps" | grep -A5 -B5 "module-name"

# Filter by specific packages
goda list "reach(./central/..., module-name/...)"

# Count unique packages using dependency
goda list "reach(./..., module-name/...)" | cut -d: -f1 | sort -u | wc -l
```

### Check version constraints

```bash
# See exact version used (works in current directory)
go list -m module-name

# Check if already updated
go list -m -u module-name

# Check for replace directives
grep "^replace.*module-name" go.mod
```

### Verify test-only usage

```bash
# List all test files importing it
find . -name "*_test.go" -exec grep -l "module-name" {} \;

# Compare with non-test files
find . -name "*.go" ! -name "*_test.go" -exec grep -l "module-name" {} \;
```

## Examples

### Example 1: Direct production dependency

**User:** "Verify where we use pgx dependency for CVE-2024-12345"

**Actions:**
1. Search go.mod: `grep pgx go.mod` → Found: `github.com/jackc/pgx/v5 v5.4.0`
2. Check usage: `grep -r "jackc/pgx" --include="*.go" --exclude-dir=vendor`
3. Use gopls: `mcp__gopls__go_search` with query "pgx"
4. Analyze files: Found in `central/database/postgres/store.go`
5. Check functions: `grep -r "QueryRow\|Query\|Exec" central/database/postgres/`

**Result:**
```
## CVE-2024-12345 Analysis: pgx SQL Injection

**Status:** USED IN PRODUCTION - HIGH PRIORITY

**Usage Summary:**
- Direct dependency: YES
- Production files: 23 files in central/
- Test files: 45 files
- Primary components: Central (PostgreSQL storage layer)

**Functionality Used:**
- CVE affects: pgx.QueryRow() with string concatenation
- We use: pgx.Query(), pgx.Exec(), pgx.QueryRow() - VULNERABLE

**Locations:**
- central/database/postgres/store.go:234 - uses QueryRow()
- central/database/postgres/migration.go:89 - uses Exec()

**Recommended Action:**
- [x] Assign to Central/Database team - CRITICAL
- [ ] Upgrade to pgx v5.4.3+ immediately

**Team Assignment:** @central-storage-team
```

### Example 2: Test-only dependency

**User:** "Is testify/assert used in production for CVE-2024-99999?"

**Actions:**
1. `grep testify go.mod` → Found as indirect
2. `grep -r "testify/assert" --include="*.go" --exclude="*_test.go"` → No results
3. `grep -r "testify/assert" --include="*_test.go"` → 500+ files

**Result:**
```
## CVE-2024-99999 Analysis: testify

**Status:** TEST CODE ONLY - LOW PRIORITY

**Usage Summary:**
- Direct dependency: NO (indirect via test dependencies)
- Production files: 0 files
- Test files: 500+ files
- Primary components: Test infrastructure only

**Recommended Action:**
- [ ] Low priority - monitor for fix
- [ ] Not production-critical

**Team Assignment:** N/A - test dependency
```

### Example 3: Transitive not-used functionality

**User:** "CVE in yaml.v3 affects Unmarshal, are we vulnerable?"

**Actions:**
1. Found yaml.v3 as transitive dependency
2. Searched actual usage: `grep -r "yaml\.Unmarshal" --include="*.go"`
3. Found we only use `yaml.Marshal` for writing config
4. CVE affects Unmarshal only

**Result:**
```
## CVE Analysis: yaml.v3 Unmarshal

**Status:** NOT VULNERABLE - Functionality not used

**Specific Functionality:**
- CVE affects: yaml.Unmarshal() parsing untrusted input
- We use: yaml.Marshal() - config writing only
- No calls to vulnerable Unmarshal function

**Recommended Action:**
- [x] Close issue - vulnerable functionality not used
- [ ] Optional: Upgrade when convenient

**Team Assignment:** N/A
```

### Example 4: Upgrade impact analysis

**User:** "We're considering upgrading to Go 1.25. What packages use old reflect APIs?"

**Actions:**
1. Search for deprecated API usage: `grep -r "reflect.SliceHeader" --include="*.go"`
2. Use gopls: `mcp__gopls__go_search` with query "SliceHeader"
3. Analyze impacted files and their component ownership
4. Check if alternatives exist

**Result:**
```
## Dependency Analysis: reflect.SliceHeader (Go 1.25 deprecation)

**Status:** PRODUCTION CODE AFFECTED - Migration needed

**Usage Summary:**
- Standard library API (deprecated in Go 1.25)
- Production files: 3 files
- Components affected: central/metrics, pkg/utils

**Locations:**
- central/metrics/collector.go:89 - uses reflect.SliceHeader for unsafe conversions
- pkg/utils/convert.go:234 - uses reflect.SliceHeader in byte slice operations

**Recommended Action:**
- [x] Migrate to unsafe.Slice() before Go 1.25 upgrade
- [ ] Review unsafe operations for alternatives

**Team Assignment:** @platform-team (pkg/utils), @central-team (metrics)
```

### Example 5: Multiple dependency analysis

**User:** "Check pgx, docker, and zap dependencies"

**Actions:**
1. Run grep searches in parallel for all three in go.mod
2. Detect replace directives: `grep "zap.*=>" go.mod`
3. Count files for each dependency
4. Identify components using each
5. Create summary table

**Result:**
```
## Multi-Dependency Analysis

### Summary Table

| Dependency | Version | Status | Production Files | Components | Priority |
|------------|---------|--------|------------------|------------|----------|
| **pgx** | v5.9.2 | Direct | 155 | Central, Scanner, Migrator | CRITICAL |
| **docker/distribution** | v2.8.3 | Direct | 3 | pkg/registries | Medium |
| **zap** | v1.18.2 (fork) | Direct | 7 | pkg/logging, operator | CRITICAL |

### Key Findings

**pgx (PostgreSQL Driver):**
- Direct dependency: YES
- Extensive usage across Scanner (primary), Central, Migrator
- 155 production files - critical for all database operations

**docker libs:**
- Limited scope: Only in pkg/registries/docker/ (3 files)
- Uses StackRox fork: heroku/docker-registry-client → stackrox/docker-registry-client
- Note: docker/distribution is EOL

**zap (Logging):**
- REPLACED: go.uber.org/zap v1.27.1 → github.com/stackrox/zap v1.18.2
- Core infrastructure: Used by entire platform via pkg/logging
- Fork is behind upstream (1.18 vs 1.27) - consider sync
```

### Example 6: Wrapper pattern (infrastructure library)

**User:** "Who needs zap logger?"

**Actions:**
1. Check go.mod: `grep zap go.mod` → Found with replace directive
2. Check indirect: `grep zap go.mod | grep indirect` → NO, it's direct
3. Find direct imports: `grep -r '"go.uber.org/zap"' --include="*.go"` → Only 7 files
4. **Detect wrapper:** All 7 files in `pkg/logging/`
5. Count wrapper usage: `grep -r 'pkg/logging"' --include="*.go"` → 563 files!
6. Run `go mod why -m go.uber.org/zap`

**Result:**
```
## Dependency Analysis: Zap Logger

**Dependency:** go.uber.org/zap v1.27.1
**Status:** WRAPPER PATTERN - Used via pkg/logging

**Usage Summary:**
- Direct dependency: YES (with StackRox fork)
- Direct zap imports: 7 files (all in pkg/logging infrastructure)
- Wrapper pattern: + 563 files via pkg/logging
- Primary components: ALL (Central, Sensor, Scanner, Operator, Migrator, roxctl, Compliance)

**Why Needed (go mod why):**
```
github.com/stackrox/rox/pkg/logging
    imports go.uber.org/zap
```

**Replace Directive:**
- Original: go.uber.org/zap v1.27.1
- Fork/Replace: github.com/stackrox/zap v1.18.2-0.20240314134248-5f932edd0404
- Version Gap: Fork at v1.18.2, upstream at v1.27.1 (9 minor versions behind)
- Reason: StackRox-specific customizations (check stackrox/zap repo)

**Wrapper Usage:**
- Wrapped by: pkg/logging
- Components using wrapper: Central (100+ files), Sensor (80+ files), ALL other components
- Architecture: All platform logging flows through pkg/logging → zap

**Recommended Action:**
- ✅ CRITICAL INFRASTRUCTURE - Foundation of all platform logging
- ⚠️ Review fork - 9 versions behind upstream
- ⚠️ Check for security patches in upstream v1.19-v1.27
- ❌ DO NOT REMOVE - Would break all logging

**Team Assignment:** @platform-team (owns pkg/logging), Impacts: ALL TEAMS
```

### Example 7: Transitive dependency

**User:** "Why do we need github.com/envoyproxy/protoc-gen-validate?"

**Actions:**
1. Check go.mod: `grep protoc-gen-validate go.mod` → Found with `// indirect`
2. **Branch to Step 2A** (transitive path)
3. Run `go mod why -m github.com/envoyproxy/protoc-gen-validate`
4. Run `go mod graph | grep protoc-gen-validate` → See chain
5. Check direct imports: `grep -r protoc-gen-validate --include="*.go"` → 0 results!
6. Find intermediate: GCS storage SDK pulls it

**Result:**
```
## Dependency Analysis: protoc-gen-validate (TRANSITIVE)

**Dependency:** github.com/envoyproxy/protoc-gen-validate v1.3.0
**Status:** TRANSITIVE ONLY - Not directly imported

**Dependency Chain:**
```
Our code (StackRox)
    ↓ imports
cloud.google.com/go/storage v1.62.1 (GCS external backup plugin)
    ↓ depends on
github.com/envoyproxy/protoc-gen-validate v1.3.0
```

**Why It Exists:**
From go mod graph:
- cloud.google.com/go/storage uses protoc-gen-validate for proto message validation
- Also pulled by: github.com/envoyproxy/go-control-plane (another transitive)

**Our Usage of Intermediate Package:**
- We import: cloud.google.com/go/storage
- Used in: 2 production files
  - central/externalbackups/plugins/gcs/gcs.go - GCS backup plugin
  - pkg/cloudproviders/gcp/utils/util.go - GCP utilities
- Components: Central (external backups)

**What Intermediate Uses It For:**
- GCS SDK uses protoc-gen-validate to generate validation code for Google Cloud API protos
- Validates proto messages in GCP service communication
- We never call protoc-gen-validate directly

**Recommended Action:**
- [x] No direct action needed - transitive dependency
- [x] Monitor for security updates (managed by cloud.google.com/go/storage)
- [ ] Low priority - updated automatically with GCS SDK

**Team Assignment:**
N/A - Managed via GCS SDK updates
(If CVE: assign to @integrations-team who owns GCS backup plugin)
```

## Troubleshooting

### Error: "goda: command not found"

**Cause:** goda not installed

**Solution:**
```bash
go install github.com/loov/goda@latest
```

### Error: "gopls not responding"

**Cause:** Large codebase, gopls indexing

**Solution:**
- Fall back to grep-based analysis
- Use `go list` and `go mod why` instead
- Wait for gopls initialization to complete

### Module path variations

**Problem:** Package imported as `pgx/v5` but searching for `pgx`

**Solution:**
- Search for base package name: `grep -i "pgx"`
- Include version suffix: `grep "pgx/v[0-9]"`
- Use go list: `go list -m all | grep pgx`

### Replace directives (forks)

**Problem:** go.mod shows one version but code uses a fork

**Example:**
```
go.uber.org/zap v1.27.1
go.uber.org/zap => github.com/stackrox/zap v1.18.2
```

**Solution:**
- Always check for replace: `grep "module-name.*=>" go.mod`
- Report BOTH versions in analysis
- Note version differences (upstream 1.27.1 vs fork 1.18.2)
- Investigate why fork exists (security patch, features, etc.)
- Consider: Should we sync with upstream?

**Impact:**
- Actual version used: The replacement version (v1.18.2)
- Declared version: What appears without replace (v1.27.1)
- CVEs may reference declared version but fork may have fix

### Too many results

**Problem:** Common package name matches everywhere

**Solution:**
- Use full import path: `grep "github.com/jackc/pgx/v5"`
- Limit to specific directories: `grep -r "pgx" central/ sensor/`
- Use goda with specific scope: `goda list "reach(./central/..., pgx/...)"`

### Uncertain about production vs test

**Quick verification:**
```bash
# Check directory structure
ls -la path/to/file.go  # Is it in /test/ or /qa-tests-backend/?

# Check imports
head -20 path/to/file.go  # Does it import testing packages?

# Check build tags
grep "//go:build" path/to/file.go  # Build constraints?
```

### Working in git worktrees

**Symptom:** Commands failing or analyzing wrong repository

**Cause:** Skill working in different directory than expected

**Solution:**
- The skill ALWAYS works in your current directory
- Do NOT change directories with `cd` commands
- If you need to analyze the main repo from a worktree, use absolute paths:
  ```bash
  grep "module-name" /path/to/main/repo/go.mod
  ```
- Use `git worktree list` to see all worktrees
- Use `pwd` to confirm current directory before analysis

**Best practice:** Stay in current worktree, analyze current go.mod

## Component to Team Mapping

When assigning issues, use this mapping based on component usage:

- **central/** → Central team (@central-team)
- **sensor/** → Sensor team (@sensor-team)
- **scanner/** → Scanner/Vulnerability team (@scanner-team)
- **roxctl/** → CLI team (@cli-team)
- **ui/** → UI team (@ui-team)
- **operator/** → Operator team (@operator-team)
- **pkg/** → Check callers, likely multiple teams
- **migrator/** → Database/Central team
- **qa-tests-backend/** → QA team, not production

## Performance Notes

- Run grep searches in parallel when possible
- Use `--include="*.go"` to avoid searching binaries
- Use `--exclude-dir=vendor` to skip vendored code
- For large repos, limit scope: `./central/...` instead of `./...`
- Cache `go mod graph` output if analyzing multiple dependencies
- When analyzing multiple dependencies, batch the go.mod greps in one message
