# Implementation Plan Template

## Structure

```markdown
# Implementation Plan: {Idea Name}

## Setup
- [ ] Verify language/runtime version and available features
- [ ] Create all scaffolding directories and files
- [ ] Install all dependencies before writing source files
- [ ] Create package/module stubs so LSP resolves imports during implementation

## Phase 1: {Phase Name} [parallel]
- [ ] Task 1: {specific, actionable description}
  - [ ] Sub Task 1: {specific, actionable description}
  - [ ] Sub Task 2: {specific, actionable description}
- [ ] Task 2: {specific, actionable description}
  - [ ] Sub Task 1: {specific, actionable description}
  - [ ] Sub Task 2: {specific, actionable description}

## Phase 2: {Phase Name}
- [ ] Task 3: {specific, actionable description}
  - [ ] Sub Task 1: {specific, actionable description}
  - [ ] Sub Task 2: {specific, actionable description}
- [ ] Task 4: {specific, actionable description}
  - [ ] Sub Task 1: {specific, actionable description}
  - [ ] Sub Task 2: {specific, actionable description}

## Dependencies
[What blocks what]

## Risks
[What could go wrong and mitigation]
```

## Quality Rules

* Always include a **Setup** phase for environment verification and dependency installation
* Mark phases/tasks as **`[parallel]`** when tasks have no cross-dependencies — this signals to the implementing agent that sub-agents can run simultaneously
* Task descriptions must name specific files, endpoints, or functions — "implement sync" is too vague, "implement `internal/sync/engine.go`: field discovery, denormalization, ALTER TABLE for new custom fields" is actionable
* For data-heavy or edge-case-heavy packages, note **"TDD recommended"** — write test fixtures and cases before implementation
* Try to keep a task sufficiently detailed for the agent. Refer to other artifacts like the spec, idea, appendix files where the additional context helps the agent

## Handoff to BeadFlow

This plan can be imported directly into beadflow for execution tracking:

```
/beadflow import sculptor {idea-name}/
```

BeadFlow reads the plan (plus spec, idea, and appendix files for context) and converts it into Beads issues automatically — no manual reformatting needed. See `beadflow/SCULPTOR-IMPORT.md` for the conversion mapping.
