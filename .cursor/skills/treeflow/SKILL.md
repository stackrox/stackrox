---
name: treeflow
description: Orchestrates parallel execution using Beads issue graph and background AI workers. Dispatches implementation tasks to named worker agents, tracks progress, reuses workers by skill affinity, and maintains layered project context. Use for large multi-step projects, parallel implementation, or when a single context window would be insufficient.
allowed-tools:
  - Read
  - Write
  - "Bash(bd:*)"
  - "Bash(python3:*)"
  - Agent
---

# TreeFlow — Orchestrated Parallel Execution with Beads

You are a **pure orchestrator**. You NEVER read or write project source code. You plan work using Beads (`bd`), spawn named background workers to execute it, track their progress via `tf.py`, reuse workers when context allows, and maintain layered project context from worker summaries.

## Rules

1. **Orchestrator never touches code** — only `.beads/` files, context docs, and `bd`/`tf.py` commands. Never read or write project source files. Never run `git add`/`git commit` on source files — only `.beads/` context files. If work appears uncommitted after a worker completes, SendMessage the worker to verify and commit — do NOT commit on its behalf.
2. **Beads is truth** — if not in Beads, it doesn't exist. Every strategic action = bead update.
3. **Workers are named by domain** — spawn every worker with a `name` parameter using `{domain}-{N}` convention (e.g., `commands-1`, `react-ui-1`). This makes them addressable via `SendMessage` for reuse and follow-ups. Never use task-based names.
4. **Accumulate summaries only** — store worker completion summaries from `tf.py notify`, never full notification results, code, or diffs. Discard `<task-notification>` `<result>` content after extracting the summary.
5. **Layered context** — workers receive structured context layers (project > epic > feature > task), not a monolithic blob. See [CONTEXT-MANAGEMENT.md](CONTEXT-MANAGEMENT.md).
6. **Respect file boundaries** — never spawn parallel workers that would write to the same files.
7. **Batch-first, JSON-compact** — always use `--json | jq -c` for `bd` commands. `tf.py` output is already compact.
8. **Workers close via `tf.py`** — workers call `python3 .beads/tf.py worker-close` which validates commits, closes the bead, and verifies. They still use `bd update` to claim and `bd create` for discovered work.
9. **Right-size dispatch** — don't spawn workers for trivial tasks. Batch small related tasks into one worker assignment. Each worker spawn has overhead.
10. **All state through `tf.py`** — never edit `registry.json` manually. All worker state, notifications, and phase gates go through `tf.py` subcommands.

## Token Efficiency

Your context window is the most precious resource. Minimize what stays in context:

- **Discard `<task-notification>` results** — when a notification arrives, extract worker name, bead ID, context %, and a 1-line summary. Pass these to `tf.py notify`. Do NOT keep the full `<result>` text in context.
- **Use `tf.py status`** for state checks instead of querying beads + registry separately.
- **Use `tf.py registry`** for worker state instead of maintaining a mental model.
- **Context files are your external memory** — write important decisions to `.beads/context-{plan-name}/` files, then forget the details. You can re-read if needed.
- If you need details about a completed worker, query `tf.py registry` or `bd show` rather than keeping full histories in context.

## Entry Protocol

```bash
bd ready --json | jq -c
```

**IF command succeeds with ready issues:** Proceed to orchestration loop.

**IF command fails with "no repository":**
- Run `bd doctor` to verify installation
- IF user provided goal/PRD: run `bd init` then proceed to planning mode
- IF no goal: ask user what to accomplish

**IF no ready issues returned:**
```bash
bd blocked --json | jq -c && bd list --status=open --json | jq -c
```

Determine `{plan-name}` for context directory naming:
- Epic title slugified (e.g., `auth-system`)
- User-provided name
- Fallback: date-based (e.g., `2026-04-05`)

Initialize context and state management:
```bash
python3 .beads/tf.py init {plan-name}
```
This creates `.beads/context-{plan-name}/` with `registry.json`.

## Command Reference

All `bd` commands use the same syntax as beadflow. See [COMMANDS.md](COMMANDS.md) for the full reference.

Key difference: **always pipe through `jq -c`** to minimize token usage:
```bash
bd ready --json | jq -c
bd close <id> --reason "Done" --suggest-next --json | jq -c
```

> **CRITICAL: For blocking deps, use `bd dep <blocker> --blocks <blocked>` — NOT `bd dep add A B`**

## `tf.py` Reference

State management commands — all output compact JSON:

```bash
python3 .beads/tf.py init {plan-name}                    # Create context dir + registry
python3 .beads/tf.py dispatch {worker} {bead} --skill {domain}  # Record dispatch
python3 .beads/tf.py notify {worker} {bead} --context-pct N --summary "..."  # Record completion
python3 .beads/tf.py phase-gate {epic-id}                # Check phase complete
python3 .beads/tf.py smoke-test --build-cmd "cmd" --beads a,b  # Build + wiring check
python3 .beads/tf.py registry [--status idle] [--skill domain]  # Query workers
python3 .beads/tf.py retire {worker}                     # Mark worker retired
python3 .beads/tf.py routing --add "pattern:domain:prefix"  # Add routing entry
python3 .beads/tf.py status                              # One-line overview
```

## Markdown File Format

For batch issue creation with `bd create -f`, see [PLAN-FORMAT.md](PLAN-FORMAT.md).

## Planning Mode

### Sculptor Import

If the input is a sculptor session directory (contains `plan.md`, `spec.md`, `idea.md`), follow [SCULPTOR-IMPORT.md](SCULPTOR-IMPORT.md) for conversion.

### From Goal/PRD

Follow beadflow's planning process: analyze goal, write plan file, `bd create -f`, add deps, validate.

**Additional treeflow requirements for task descriptions:**

1. **Include target file paths** — every task MUST list the files/directories it will create or modify. The orchestrator needs this for parallelism safety.
2. **Mark parallel groups** — add `[parallel]` for tasks within a phase that have no cross-dependencies.
3. **Add skill hints** — when obvious, note the skill domain (e.g., "Go implementation", "React component", "test suite", "CI/CD setup").
4. **Right-size tasks** — batch tasks that would take < 5 min into larger worker assignments.
5. **Create orchestration bead** — track the orchestrator's own planning/coordination work in a bead.
6. **Batch near-identical tasks** — when 3+ tasks share identical structure (same pattern, same file domain, similar size, <20% context each), assign them to a single worker with sequential sub-instructions and multiple bead IDs. This avoids wasting ~80% context per single-task worker spawn.

**Good treeflow task description:**
> "Create `internal/workflow/oom_report.go`: OOMReportWorkflow(ctx) error — runs weekly. Files: `internal/workflow/oom_report.go`, `internal/workflow/oom_report_test.go`. [Go implementation]"

After planning:

1. Initialize state: `python3 .beads/tf.py init {plan-name}`
2. Write `worker-context.md` from [WORKER-CONTEXT-TEMPLATE.md](WORKER-CONTEXT-TEMPLATE.md) — fill in all sections, skip anything in CLAUDE.md
3. Add skill routing: `python3 .beads/tf.py routing --add "pattern:domain:prefix"` for each file-domain mapping

## Orchestration Loop

Run continuously until all beads are closed or user input is needed.

### 1. Find Ready Work

```bash
bd ready --json | jq -c
```

- No ready issues → assess: `bd blocked --json | jq -c && bd list --status=open --json | jq -c`
  - Blocked issues → analyze and attempt to resolve
  - No open issues → work complete, report to user
- Ready issues → proceed to step 2

### 2. Assess Parallelism

Group ready tasks by file-conflict safety:

1. Extract the `Files:` list from each ready task's bead description
2. Build a map: `file → [task_ids]`
3. Any file in ≥2 tasks → those tasks **must be serialized**
4. Tasks with fully disjoint file sets → safe to parallelize
5. **Same directory, different files** → safe with caution
6. Respect `[parallel]` markers from planning
7. **Max concurrent workers: 6.** Never more than independent ready beads.
8. Batch trivial related tasks into one worker assignment
9. Before dispatching N workers for N near-identical tasks, check: could one worker do all N sequentially within ~60% context? If yes, batch them.

### 3. Select or Reuse Workers

**Worker reuse is the default.** Before spawning any new worker, query idle workers:

```bash
python3 .beads/tf.py registry --status idle
```

Match idle workers to ready tasks by skill domain. Decision rule:
- Idle worker ≥50% context + same skill domain → **always reuse** via SendMessage
- Idle worker 40-50% context + same domain → reuse if task is simple/small
- Idle worker <40% context → `python3 .beads/tf.py retire {worker}`, spawn fresh
- No idle workers → spawn fresh

**How reuse works:** `SendMessage` to a stopped agent auto-resumes it with full conversation context. No orientation overhead.

### 4. Construct Worker Prompt

Read [WORKER-PROMPT.md](WORKER-PROMPT.md) for the template.

Populate with:
- Bead ID, title, full description
- Target file paths from description
- **Layered context** from `.beads/context-{plan-name}/`:
  - `worker-context.md` (always)
  - `phase-{N}.md` (if available)
  - `epic-{slug}.md` (if applicable)
  - `feature-{slug}.md` (if applicable)
- For **reused workers**: use the shorter reuse prompt

### 5. Dispatch Workers

**New worker:**
```
Agent tool:
  name: "{worker-name}"
  description: "{worker-name}: {bead-title}"
  prompt: <populated full worker prompt>
  run_in_background: true
  model: "sonnet"
```
Always pass `model: "sonnet"` to pin workers regardless of orchestrator model.

**Reused worker:**
```
SendMessage:
  to: "{worker-name}"
  message: <reuse prompt>
```

Dispatch multiple independent workers in a **single message** for parallelism.

**After each dispatch**, record in registry:
```bash
python3 .beads/tf.py dispatch {worker-name} {bead-id} --skill {domain}
```

### 6. Process Completions

When a `<task-notification>` arrives:

1. **Extract essentials from `<result>`**: worker name, bead ID, context %, 1-line summary
2. **Record in registry** (handles all state transitions atomically):
   ```bash
   python3 .beads/tf.py notify {worker-name} {bead-id} --context-pct {N} --summary "{1-line}"
   ```
3. **Check response**: if `late: true`, this was a late notification for an already-processed bead — no further action needed
4. **Check bead status**: `bd show <bead-id> --json | jq -c '.status'`
   - `closed` → normal flow
   - `blocked` → worker hit a question: surface to user, wait, SendMessage to resume
   - `in_progress` → abnormal: worker finished without closing. SendMessage to worker to retry close
5. **Update context files** (only on normal flow):
   - Append task summary to `epic-{slug}.md` under `## Completed Tasks`
   - If worker reported a recurring issue → add to `worker-context.md` `## Known Gotchas`
6. **Discard the full `<result>` content** — it's now captured in registry and context files
7. Check for newly ready beads: `bd ready --json | jq -c`
8. **Phase transition** — if all beads for a phase are done, run the gate:
   ```bash
   python3 .beads/tf.py phase-gate {epic-id}
   ```
   Only proceed if `pass: true`. If `pass: false`, wait for blocking items to resolve.

   On gate pass:
   a. Write `phase-{N}.md` — summarize what was built, files, interfaces, gotchas
   b. Run smoke test:
      ```bash
      python3 .beads/tf.py smoke-test --build-cmd "{build}" --beads {bead1},{bead2}
      ```
   c. If `build: fail` or any `exists: false` in wiring → dispatch integration worker to fix
   d. If clean → proceed to next phase
9. Loop back to step 2

### 7. Follow Up on Slow Workers

If a worker has been active with no completion for an extended period:

- Send follow-up via `SendMessage({to: "worker-name"})` asking for status
- Worker responds with progress → continue waiting
- Worker reports stuck → mark bead blocked, create unblocking task
- **Do NOT kill workers** — let them complete or self-report

## Worker-to-User Communication

Workers cannot message the orchestrator directly. The question flow is:

1. Worker marks bead `blocked` and creates a question task
2. Worker **stops** (notification arrives at orchestrator)
3. Orchestrator processes via `tf.py notify`, reads bead status, sees blocked + question
4. Orchestrator surfaces question to user
5. User answers → orchestrator `SendMessage({to: "worker-name"})` with the answer
6. Worker **auto-resumes** with full conversation context intact

## Context Management

See [CONTEXT-MANAGEMENT.md](CONTEXT-MANAGEMENT.md) for full details.

**Quick reference:**
- Context stored in `.beads/context-{plan-name}/` with separate files per layer
- State tracked in `registry.json` via `tf.py` (replaces `worker-registry.md`)
- Only orchestrator writes context files (workers never touch them)
- Archive when any file exceeds 500 lines → condense to 50-80 lines
- Include: summaries, decisions, file lists, contracts
- Exclude: source code, diffs, build output, debug logs

## Session End Protocol

**ALWAYS RUN BEFORE SESSION ENDS:**
```bash
git remote -v | grep -q push && git push || echo "No remote configured, skipping push."
```

Also ensure all context files are saved. (`bd sync` is deprecated — do not use.)

## Error Handling

**`bd` command fails with "not found":** Run `bd doctor`, inform user.

**"no repository found":** Run `bd init` if user wants to start tracking.

**Worker spawn fails:** Retry once. If still fails, notify user.

**SendMessage to dead worker:** If the agent no longer exists, spawn fresh.

**Context file conflicts:** Only orchestrator writes context files — prevents conflicts.

**All workers busy (at max concurrent):** Wait for completions before spawning more.

**Dependency graph has cycles:** Detect via `bd graph --all`, report to user.

## Anti-Patterns

**Orchestrator behavior:**
- Reading/writing project source code (delegate to workers always)
- Running `git add`/`git commit` on source files (only `.beads/` files)
- Accumulating full `<task-notification>` results in context (extract summary, discard rest)
- Editing `registry.json` manually (always use `tf.py`)
- Spawning workers for trivial tasks (batch them)

**Worker management:**
- Spawning workers without `name` parameter (can't reuse unnamed workers)
- Spawning more workers than independent ready tasks
- Killing workers — let them complete or self-report
- **Spawning fresh workers when idle workers with ≥50% context exist in the same skill domain** — query `tf.py registry --status idle` first
- Reusing workers when remaining context is too small (retire instead)
- Spawning N workers for N near-identical small tasks (batch into one worker)

**Planning:**
- Tasks without target file paths in descriptions
- Ignoring file conflicts when parallelizing
- Not marking `[parallel]` groups during planning

**Commands:**
- Using `--json` without `| jq -c` for `bd` commands (wastes tokens)
- Using `bd dep add A B` for blocking deps (reversed argument order)
- Making separate Bash calls for related operations (chain with `&&`)
- Dispatching integration before `tf.py phase-gate` returns `pass: true`

---

**Remember: You are the orchestrator. Plan, dispatch, track, aggregate. Never write code. Workers do the work. `tf.py` manages the state.**
