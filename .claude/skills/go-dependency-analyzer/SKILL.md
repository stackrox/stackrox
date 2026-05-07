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

## Prerequisites

This skill requires the `goda` tool for dependency graph visualization. Install it with:

```bash
go install github.com/loov/goda@latest
```

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

A dependency can be BOTH direct and indirect (imported directly by some packages, pulled transitively by others).

Check if dependency is marked `// indirect`:
- **If YES (only indirect):** Follow **Step 2A: Transitive Dependency Analysis Path**
- **If NO (direct, or both direct+indirect):** Follow **Step 3: Direct Dependency Analysis Path**

Note: If `go mod why` shows direct import paths AND the dependency is marked `// indirect`, it means the dependency is used both ways - report this in the analysis.

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

**Check for wrapper pattern:** Some dependencies are wrapped by `pkg/` packages (e.g., zap via pkg/logging). If wrapped, count BOTH direct imports AND wrapper usage.

```bash
# Find direct imports
grep -r '"module-name' --include="*.go" --exclude-dir=vendor | cut -d: -f1 | sort -u | head -20
goda list "reach(./..., module-name/...)" | head -30

# Separate production vs test
grep -r '"module-name' --include="*.go" --exclude="*_test.go" --exclude-dir=vendor | wc -l
grep -r '"module-name' --include="*_test.go" --exclude-dir=vendor | wc -l

# If wrapper detected in pkg/, count wrapper users too
grep -r 'pkg/wrapper"' --include="*.go" --exclude="*_test.go" --exclude-dir=vendor | wc -l
```

### Step 5: Analyze Used Functionality (Direct Dependencies Only)

Use gopls MCP tools (`mcp__gopls__go_search`, `mcp__gopls__go_symbol_references`) or grep to find specific function usage:

```bash
# Search for vulnerable functions mentioned in CVE
grep -r "FunctionName" --include="*.go" --exclude-dir=vendor | head -20
```

Note: GitNexus hooks may auto-provide symbol context if available.

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

**Team Assignment:**
[Team name(s) from CODEOWNERS based on component usage]
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

**Team Assignment:**
N/A - Transitive dependency managed via [intermediate-package] updates
[If CVE with production impact: assign to team(s) from CODEOWNERS owning files that use the intermediate package]
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

**Team Assignment:**
@stackrox/core-workflows (from CODEOWNERS: `pkg/postgres/**/*` and `migrator/**/*`)
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

**Team Assignment:**
N/A - Test-only transitive dependency (not production-critical)
```

### Example 3: Multiple dependency analysis

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

**Team Assignment:**
From CODEOWNERS: Multiple teams own components using logging.
Primary: Team owning `pkg/logging` wrapper.
Impact: ALL TEAMS (critical infrastructure)
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

When assigning issues, consult `.github/CODEOWNERS` to determine team ownership based on the files/directories where the dependency is used.

**How to use CODEOWNERS:**
1. Identify which files/directories use the dependency (from your analysis)
2. Read `.github/CODEOWNERS` to find matching patterns
3. The LAST matching line wins (CODEOWNERS rule)
4. Assign to the team(s) listed for those patterns

**Multi-team dependencies:** If used across multiple components with different owners, list all affected teams.

## Performance Notes

- Run grep searches in parallel when possible
- Use `--include="*.go"` to avoid searching binaries
- Use `--exclude-dir=vendor` to skip vendored code
- For large repos, limit scope: `./central/...` instead of `./...`
- Cache `go mod graph` output if analyzing multiple dependencies
- When analyzing multiple dependencies, batch the go.mod greps in one message
