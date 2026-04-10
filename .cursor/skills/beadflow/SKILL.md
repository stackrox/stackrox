---
name: beadflow
description: Autonomous task planning and execution using Beads (bd). Use when working on multi-step projects, breaking down PRDs or specs into tasks, managing complex implementations, or tracking progress on development work. Creates and manages issues in the Beads issue graph.
allowed-tools: "Read, Write, Bash(bd:*)"
---

# BeadFlow - Autonomous Planning & Execution with Beads

You are an autonomous agent using **Beads** (`bd`) as the system of record. Every strategic action must create or update a Beads issue.

## Rules

1. **Beads is truth** - If not in Beads, it doesn't exist. Never do work without a corresponding issue.
2. **Always update** - Every action = Beads update. Done = close. Blocked = mark blocked + comment why.
3. **Small units** - Tasks must be completable in one session. Decompose anything larger.
4. **Proper types** - Use correct issue types for hierarchy (epic > feature > task).
5. **Durable issues** - Write so another agent can resume without conversation context.
6. **Batch-first** - Prefer batch commands over individual calls. Use `bd create -f` for multiple issues. Use multi-ID updates/closes. Chain commands with `&&`.
7. **JSON output** - Always use `--json` flag for structured, machine-readable output.

## Entry Protocol

Run on skill activation:

```bash
bd ready --json
```

**IF command succeeds:** Proceed to the execution loop with the returned ready issues.

**IF command fails with "no repository":**
- Run `bd doctor` to verify installation
- IF user provided goal/PRD: run `bd init` then proceed to planning mode
- IF no goal: ask user what to accomplish

**IF no ready issues returned:**
- Run `bd blocked --json && bd list --status=open --json` to assess state

## Type Selection

| Type | Use When | Priority Default |
|------|----------|------------------|
| `epic` | Top-level goal, major deliverable | P0 |
| `feature` | User-facing capability, delivers user value | P1 |
| `task` | Implementation work, concrete action | P2 |
| `bug` | Defect, something broken | P1 |
| `chore` | Refactor, cleanup, no user-facing change | P3 |

**Decision logic:**
- "What should user see?" → `feature`
- "How do we build that?" → `task`
- "Top-level goal?" → `epic`
- "Broken?" → `bug`
- "Cleanup/refactor?" → `chore`

## Priority Scale

- `0` (P0/CRITICAL) - Blocks everything, drop all other work
- `1` (P1/HIGH) - Important features, major bugs
- `2` (P2/MEDIUM) - Standard work
- `3` (P3/LOW) - Nice-to-have
- `4` (P4/BACKLOG) - Future, not planned

## Command Reference

See [COMMANDS.md](COMMANDS.md) for the full command reference including batch creation, updates, closes, dependencies, comments, visibility, and command chaining.

Key commands:
```bash
bd ready --json                    # Find unblocked, actionable issues
bd update <id> --status in_progress --json  # Claim work
bd close <id> --reason "Done" --suggest-next --json  # Close and get next
bd dolt push                       # ALWAYS run before session end
```

> **CRITICAL: For blocking deps, use `bd dep <blocker> --blocks <blocked>` — NOT `bd dep add A B`** (argument order is unintuitive).

## Markdown File Format

For batch issue creation with `bd create -f`, see [PLAN-FORMAT.md](PLAN-FORMAT.md) for the full format specification and examples.

## Planning Mode

### Sculptor Import

If the input is a sculptor session directory (contains `plan.md`, `spec.md`, `idea.md`), follow [SCULPTOR-IMPORT.md](SCULPTOR-IMPORT.md) for conversion mapping and enrichment steps. This avoids manual reformatting — beadflow reads the sculptor artifacts and generates the bd-compatible plan automatically.

### From Goal/PRD

When user provides goal/PRD and `.beads/` is initialized:

### 1. Analyze the Goal
Read and understand the PRD/goal. Identify the epic, features, and tasks. No bash calls needed.

### 2. Write the Plan File
Use the Write tool to create a `.md` file with all issues:

```bash
# Agent writes: .beads/plan.md (using Write tool)
# Contains all epics, features, and tasks in markdown format
```

**Planning principles:**
- Epic = "Goal: X" format, describes end state
- Features = user-facing capabilities
- Tasks = concrete, actionable work (specific files, endpoints, functions)
- Name by WHAT (deliverable), not WHEN (timeline)
- Each task = 1 focused session max
- **Always include a Setup task** as the first task (verify versions, install deps, create stubs)
- **Always include a Coverage Review task** as the last task in each phase: *"List all [activities/handlers/modules] in this phase with no test coverage and justify each omission."* This makes coverage tradeoffs explicit before they accumulate into debt.
- **Mark parallel groups** — add `[parallel]` in descriptions for tasks within a phase that have no cross-dependencies. This signals that sub-agents can execute them simultaneously.
- **Flag TDD candidates** — add `[TDD]` for data-heavy or edge-case-heavy tasks where writing test fixtures first prevents bugs (e.g. parsers, denormalizers, format converters)
- **Add a shared-dependencies task** for any phase that introduces a central registry, dependency container, or shared struct (e.g. an `Activities` struct, a service locator, a plugin registry). This task runs first in the phase and stubs all fields/dependencies from the spec's external systems list. Files written later in the phase reference the stub; the build confirms wiring. Without this, every new client/dependency causes a retroactive edit to the shared struct scattered across multiple commits.

**Good task examples:**
- "Create User model with email, password_hash, created_at fields in models/user.py"
- "Add POST /api/auth/login endpoint in routes/auth.py returning JWT"
- "Write unit tests for authenticate() in tests/test_auth.py"

The highest-quality task descriptions include: file path + function name + signature + behavior + thresholds + destination. Example:
> "Create `internal/workflow/oom_report.go`: OOMReportWorkflow(ctx) error — runs weekly. AggregateOOMEvents(ctx) ([]OOMEvent, error) — query Konflux PipelineRun logs. If count > threshold (5/week): post to #konflux-users."

This format is executable without any further design work and produces correct first-draft code consistently.

**Bad task examples:**
- "Implement backend" (too vague)
- "Handle auth" (unclear scope)
- "Do the database stuff" (not actionable)

### 3. Batch Create All Issues
```bash
bd create -f .beads/plan.md --json
```
One command creates all issues. Parse the JSON output for ID mappings.

### 4. Add Dependencies
Chain all dependency additions in one or two calls:
```bash
# Cross-issue blocking dependencies — use --blocks (blocker first, reads naturally):
bd dep <task-a> --blocks <task-b> && bd dep <task-b> --blocks <task-c> && ...

# Parent-child hierarchy — use dep add with -t parent-child (child first, parent second):
bd dep add <task> <feature> -t parent-child && bd dep add <feature> <epic> -t parent-child && ...
```

**NEVER use `bd dep add A B` for blocking** — the argument order (`blocked` first, `blocker` second) is unintuitive and causes reversed graphs. Always use `bd dep <blocker> --blocks <blocked>` for blocking relationships.

### 5. Validate
```bash
bd ready --json
```
Should show at least one actionable task. Run `bd graph --all` if structure needs visual verification.

**Total for 50 issues: ~4-6 tool calls** (1 Write + 1 create + 1-3 dep chains + 1 validate)

## Execution Loop

Run continuously until no ready issues or user input needed:

### 1. Find Work
```bash
bd ready --json
```
The `--json` output includes full issue details (title, description, priority, dependencies). No separate `bd show` call needed.

**IF no issues returned:**
- Run `bd blocked --json && bd list --status=open --json` to assess state
- IF blocked issues exist: analyze and resolve blockers
- IF no open issues: work is complete, report to user

**IF issues returned:**
- Select highest priority (lowest number)
- Proceed to step 2

### 2. Claim and Execute
```bash
bd update <id> --status in_progress --json
```

Execute work:
- Do EXACTLY what issue describes, no scope creep
- Do NOT add features, refactor unrelated code, or "improve" things
- Stay focused on single issue completion criteria

**Before writing any new type, class, or struct** — search the codebase for existing definitions first. Duplicate type definitions cause compile errors that require a read-fix cycle. One grep before writing saves multiple round-trips:
```bash
# Search for existing type definitions before creating a new one
grep -r "type TypeName\|class TypeName\|TypeName =" ./src/
```

**Before writing any call site** — verify the callee's actual signature with LSP hover or grep, not memory. Wrong signatures (wrong arg count, wrong constant name, wrong method name) are a common error class that `go build` catches but requires a read-fix-rebuild cycle to resolve. The CLAUDE.md rule "prefer LSP" applies at authoring time, not just navigation time.

**After writing or editing any file** — use the compiler/build tool as ground truth, not LSP diagnostics. LSP diagnostics on recently-modified files can lag and show stale errors from the previous version. `go build ./...`, `cargo check`, `tsc`, etc. are authoritative.

**When writing a stub** — mark it explicitly and document the contract:
```go
// STUB: real implementation calls POST /api/v1/clusters with ClusterConfig JSON.
// Expected response: {"name": "cluster-name", "status": "provisioning"}
// Error conditions: 409 if name already exists, 403 if quota exceeded.
func (c *Client) CreateCluster(ctx context.Context, cfg ClusterConfig) (string, error) {
    return cfg.Name, nil
}
```
The function body is a placeholder; the comment is the real value. A reader implementing the production version should not have to infer the API contract from context.

**Parallel execution:** When multiple ready issues are independent (marked `[parallel]` or no cross-dependencies), claim them all and use the `Agent` tool to run sub-agents in parallel. Each sub-agent gets one task. This can be 3-4x faster for phases like initial package implementation.

**TDD for complex logic:** For tasks marked `[TDD]` or involving data transformation, parsing, or edge-case-heavy logic:
1. Write test fixtures (sample inputs in `testdata/` or inline)
2. Write test cases with **computed** expected values (don't guess — actually calculate the expected output)
3. Implement until tests pass
This catches bugs that post-hoc testing misses, especially for format conversions and denormalization.

### 3. Handle Outcome

**IF work completed successfully:**
```bash
bd close <id> --reason "Summary of what was done" --suggest-next --json
```
The `--suggest-next` flag returns the next ready issue. Continue from step 2 with the suggested issue.

**IF blocked (need API key, external dependency, user decision):**
```bash
bd update <id> --status blocked --json && bd create "Unblock: <what's needed>" -t task -p 1 --deps "<blocked-id>" -d "<how to resolve>" --json
```
One chained call handles: mark blocked + create unblocking task with dependency. Return to step 1.

**IF discovered new work during execution:**
```bash
bd create "Found: <new thing>" -t task -p 2 --deps "discovered-from:<current-id>" -d "<what needs doing>" --json
```
One call handles: create + link provenance. Continue current work.

**IF issue too large (will take >1 session):**
```bash
bd create "Subtask 1: <specific part>" -t task --parent <large-id> -d "..." --json && bd create "Subtask 2: <specific part>" -t task --parent <large-id> -d "..." --json && bd close <large-id> --json
```
One chained call handles: create subtasks + close parent. Return to step 1.

## State Detection & Actions

### When `bd ready` returns empty
```bash
bd blocked --json && bd list --status=open --json && bd list --status=in_progress --json
```
One chained call to assess full state:
1. Blocked issues? → focus on unblocking
2. No open issues? → work complete
3. Stale in-progress items? → check if you should resume or close them
4. All clear? → report completion to user

### When encountering errors in work
- DO NOT immediately mark blocked
- Attempt to resolve (check code, read docs, fix issues)
- ONLY mark blocked if truly cannot proceed without external input

### When user provides new goal mid-session
- Complete current issue or leave in_progress (don't abandon)
- Create new epic for new goal
- Ask user if they want to switch focus or finish current work first

## Session End Protocol

**ALWAYS RUN BEFORE SESSION ENDS:**
```bash
bd dolt push
```
This persists Beads state to git. Without this, changes may not sync to remote. (`bd sync` is deprecated — do not use it.)

## Error Handling

**IF `bd` command fails with "not found":**
- Run `bd doctor` to check installation
- Inform user Beads not installed or not in PATH

**IF command fails with "no repository found":**
- Run `bd init` if user wants to start tracking
- Confirm before initializing

**IF `bd create -f` fails with format error:**
- Check that the file uses `## Title` (H2) for issues and `### Section` (H3) for metadata
- Use `bd create -f plan.md --dry-run --json` to validate before committing

**IF dependency graph has cycles:**
- Detect via `bd graph --all` output
- Report to user, ask which dependency to remove

## Anti-Patterns (DO NOT DO)

- Creating issues without executing them (plan paralysis)
- Working without claiming issue first (no audit trail)
- Closing issues that aren't actually done (false progress)
- Creating mega-tasks that take multiple sessions (decompose first)
- Adding "nice to have" scope to existing issues (create separate issue)
- Forgetting `bd dolt push` at session end (sync failure) — `bd sync` is deprecated, do not use it
- **Using individual `bd create` calls to plan multiple issues** (use `bd create -f` instead)
- **Using `bd show` after `bd ready --json`** (JSON output already includes full details)
- **Closing then calling `bd ready` separately** (use `bd close --suggest-next` instead)
- **Omitting `--json` flag** (human-readable output is harder to parse and wastes tokens)
- **Making separate Bash calls for related operations** (chain with `&&` instead)
- **Using `bd dep add A B` for blocking deps** — argument order is `<blocked> <blocker>` (reversed from intuition). Use `bd dep <blocker> --blocks <blocked>` instead.
- **Writing source files before installing dependencies** — install all deps first so LSP works cleanly from the start
- **Implementing independent tasks sequentially** when sub-agents could run them in parallel — use the `Agent` tool for `[parallel]` task groups
- **Guessing test expected values** — compute the correct expected output; wrong assertions waste debug cycles (e.g. Levenshtein distance, hash values)
- **Inconsistent naming across files** — pick one name (`jsonErr` or `jsonError`, not both) and use it everywhere from the start
- **Defining types without checking for existing ones** — always search the codebase before writing a new type/class/struct; duplicate definitions cause compile errors
- **Writing call sites from memory** — always verify a function's actual signature with LSP hover or grep before calling it; never rely on what you think the signature "should" be
- **Trusting LSP diagnostics on recently-edited files** — LSP can lag after writes; use the compiler/build tool as the authoritative check
- **Writing stubs without documenting their contract** — a stub with no comment forces the next implementer to reverse-engineer the expected API; always document what the real implementation should do
- **Building a central registry/struct incrementally** — if a phase introduces a shared dependency container (service locator, activity struct, plugin registry), enumerate ALL required dependencies from the spec at the start of the phase, stub them as nil/empty, then write individual implementation files that reference them; avoids scattered retroactive edits
- **Creating helpers for one-liners** — if the "helper" is two words in the standard library (e.g. `fmt.Errorf`), don't abstract it; the abstraction adds indirection without value

---

**Remember: Batch-first. JSON always. Chain commands. If it's not in Beads, it doesn't exist. If it's ready, work it.**
