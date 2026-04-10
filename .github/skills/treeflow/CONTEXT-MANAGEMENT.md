# Context Management — Layered Context System

Workers receive structured context layers relevant to their task, not a monolithic document. This keeps worker prompts focused and avoids wasting context window on irrelevant information.

## Context Layers

| Layer | Scope | Content | Who Gets It |
|-------|-------|---------|-------------|
| **Project** | All workers | Tech stack, repo structure, specs, conventions | Everyone |
| **Epic** | Same epic | Epic goal, architecture decisions, completed features | Epic workers |
| **Feature** | Same feature | Feature spec, API contracts, related task summaries | Feature workers |
| **Task** | Single worker | Bead description, target files, specific instructions | Assigned worker |

## Storage Structure

```
.beads/context-{plan-name}/
├── worker-context.md   # Sent to workers: tech stack, conventions, known gotchas
├── registry.json       # Orchestrator-only: worker state, routing, phases (managed by tf.py)
├── phase-1.md          # Written after Phase 1 closes (summary for Phase 2+ workers)
├── phase-2.md          # Written after Phase 2 closes
├── epic-{slug}.md      # Per-epic context
├── feature-{slug}.md   # Per-feature context (optional)
└── archive/            # Archived context files
    ├── worker-context-v1.md
    └── epic-auth-v1.md
```

## Worker Context (`worker-context.md`)

Sent to all workers. Created by the orchestrator after planning. Contains:

- Tech stack and repo structure overview
- Key specs/PRD summary (not full text — link to source files)
- Coding conventions and patterns established in the project
- Shared infrastructure decisions (e.g., "using PostgreSQL", "monorepo with Go modules")
- **Known Gotchas** (updated as workers discover recurring issues)

**Do not duplicate content already in `CLAUDE.md`** — workers receive `CLAUDE.md` automatically. Only add project-specific context that `CLAUDE.md` doesn't cover.

**All workers receive this layer.**

## Worker Registry (`registry.json`)

**Orchestrator-only — never sent to workers.** Managed exclusively by `tf.py` — never edit manually.

Contains worker state (status, context %, last bead, notification tracking), skill routing table, and phase gate tracking. See [WORKER-REGISTRY-TEMPLATE.md](WORKER-REGISTRY-TEMPLATE.md) for the full JSON schema.

**Context % source of truth:** Always use the worker's self-reported `context_pct` value as recorded by `tf.py notify` or `tf.py worker-close`. The orchestrator does not estimate context usage — it reads from `tf.py registry`.

Key commands:
```bash
python3 .beads/tf.py registry --status idle    # Find reuse candidates
python3 .beads/tf.py status                    # One-line overview
python3 .beads/tf.py phase-gate {epic-id}      # Check phase completion
```

## Phase Summary Files (`phase-{N}.md`)

Written by the orchestrator after all beads for Phase N close. Sent to Phase N+1 and later workers as an additional context layer.

**Content:**
```markdown
# Phase N Summary

**Status:** Complete | **Build:** Passing

## Files Created/Modified
- `lib/engine/parser.ts` — ParsedPipeline, ParsedCommand interfaces
- `lib/engine/commands/ls.ts` — LsCommand
...

## Key Interfaces Exposed
{TypeScript interfaces that Phase N+1 workers need to know about}

## Integration Points
{What Phase N+1 must wire up — e.g., "register commands in background.ts"}

## Known Gotchas Discovered
{e.g., "TS 'Cannot find name chrome' is an LSP false positive in WXT projects — build still passes"}

## Build Status
{Output from build/typecheck at phase end}
```

Phase summaries capture **learned context** that doesn't survive orchestrator restarts. They complement beads state (which tracks WHAT is done) with discovered knowledge (HOW it was done and what to watch for).

## Epic Context (`epic-{slug}.md`)

Per-epic file. Created when the first task in an epic is dispatched. Contains:

- Epic goal and acceptance criteria (from bead description)
- Architecture decisions made during this epic
- Completed feature summaries (appended as features finish)
- Cross-cutting concerns discovered during implementation

**Workers on this epic receive this layer.**

## Feature Context (`feature-{slug}.md`)

Optional, per-feature. Created when a feature has multiple tasks that need shared context. Contains:

- Feature spec / API contracts
- Data models and interfaces
- Related task summaries (appended as tasks complete)

**Workers on this feature receive this layer.**

## Updating Context After Worker Completion

When a worker completes and closes its bead:

1. Record completion via `tf.py notify` (updates registry atomically)
2. Extract summary from notification and append to the appropriate context layer:
   - Task summary → epic context file (under `## Completed Tasks`)
   - Feature-level decisions → feature context file
   - Project-level decisions → `worker-context.md`
   - Recurring "issues that aren't issues" → `worker-context.md` under `## Known Gotchas`
3. **Discard the full `<task-notification>` result** — it is now captured in registry and context files

### Summary Format

```markdown
### BD-{id}: {title}
**Worker**: {worker-name} | **Files**: {files created/modified}
{Summary from close reason — 2-5 lines}
```

## Archival

When any context file exceeds **500 lines**:

1. Move current file to `archive/{filename}-v{N}.md`
2. Create fresh file with condensed summary:
   - **Worker context**: 50-80 lines covering current state, key decisions, active conventions, skill routing
   - **Epic context**: 30-50 lines covering goal, architecture, completed work summary
   - **Feature context**: 20-30 lines covering spec, interfaces, completed tasks
   - **Phase files**: do not archive — they are fixed-size summaries, not accumulating logs
3. New workers receive only the current (condensed) version
4. Orchestrator can archive proactively if context is growing fast

## What Goes In vs Out

| Include | Exclude |
|---------|---------|
| Worker summaries (from `bd close --reason`) | Full source code |
| Key implementation decisions with rationale | Complete diffs or patches |
| File lists (created/modified) | Build output or logs |
| Architecture choices | Debug traces |
| API contracts and interfaces | Intermediate failed attempts |
| Convention changes | Tool output (test results, lint) |

## Orchestrator-Only Writes

Only the orchestrator writes to context files. Workers never touch them. This prevents:
- Race conditions between parallel workers
- Inconsistent context state
- Workers accidentally overwriting each other's summaries
