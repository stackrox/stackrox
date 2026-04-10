# Importing Sculptor Plans

When a sculptor session produces a plan (`plan.md`), beadflow can import it directly — no manual reformatting needed.

## Entry

```
/beadflow import sculptor <path-to-idea-directory>
```

Example: `/beadflow import sculptor rtk-v2/`

## What to Read

Read all sculptor artifacts in the directory for context. Priority order:

1. `plan.md` — primary input, defines tasks and phases
2. `spec.md` — enriches task descriptions with implementation details
3. `idea.md` — provides problem context and chosen approach
4. `research.md` + `appendix-*.md` — background context if task descriptions need clarification

## Conversion Mapping

### Structure Mapping

| Sculptor Plan | Beads Issue | Notes |
|---|---|---|
| Plan title (`# Implementation Plan: X`) | `epic` (P0) | Becomes the top-level epic |
| Phase heading (`## Phase 1: Name`) | Not an issue | Phases are grouping/ordering, not issues |
| Task (`- [ ] Task: description`) | `task` (P2) | Or `feature` if user-facing |
| Sub-task (`  - [ ] Sub Task: description`) | `task` (P2) | Child of parent task |
| `## Setup` phase | `task` (P1) | Single setup task, always first |
| `## Dependencies` section | `bd dep` calls | Explicit dependency info |
| `## Risks` section | Comment on epic | Preserved as context, not separate issues |

### Markers

| Sculptor Marker | Beads Handling |
|---|---|
| `[parallel]` on phase | Tasks within phase get NO cross-dependencies |
| `[TDD]` on task | Add `[TDD]` to task description + label `tdd` |
| `TDD recommended` in quality rules | Same as `[TDD]` |

### Dependency Rules

1. **Within a `[parallel]` phase**: no blocking deps between tasks
2. **Within a sequential phase** (no `[parallel]`): tasks block in order (task 1 → task 2 → ...)
3. **Between phases**: last task(s) of phase N block first task(s) of phase N+1
4. **Sub-tasks**: parent-child relationship to their parent task. Sub-tasks block their parent's completion.
5. **Explicit `## Dependencies` section**: override/supplement the above with any explicitly stated deps
6. **Setup phase**: always unblocked, blocks everything in Phase 1

### Enriching Descriptions

The sculptor plan often has terse task names. Enrich them using the spec:

- If `spec.md` has an exact schema for a task's data model → include it in the task description
- If `spec.md` has API contracts for a task's endpoint → include request/response shapes
- If `appendix-*.md` has sample payloads relevant to a task → reference the appendix file path
- If `idea.md` explains the reasoning behind an approach → add a one-line "Context:" note

Keep descriptions self-contained enough that an agent can execute without re-reading all sculptor artifacts.

## Conversion Steps

### 1. Read sculptor artifacts
Read `plan.md`, then `spec.md` and `idea.md` for context.

### 2. Generate `.beads/plan.md`
Convert using the mapping above. Write in [bd create -f format](PLAN-FORMAT.md):

```markdown
## Goal: {Plan Title}

### Type
epic

### Priority
0

### Description
{From idea.md problem statement + chosen approach}

## {Task name from plan}

### Type
task

### Priority
2

### Description
{Enriched description from spec + plan context}
```

### 3. Create and wire up
```bash
bd create -f .beads/plan.md --json
```

Then add dependencies per the rules above. See [COMMANDS.md](COMMANDS.md) for dependency syntax.

### 4. Validate
```bash
bd ready --json
```

Setup task should be the first (or only) ready issue.
