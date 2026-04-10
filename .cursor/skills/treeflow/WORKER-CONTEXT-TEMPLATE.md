# {Plan Name} — Worker Context

> Sent to all workers. Do not include Worker Registry or Skill Routing — those are orchestrator-only (registry.json).
> Remove this header line before writing the file.

## Overview

{1-2 sentence description of what is being built and why}

## Tech Stack

- **Language/Runtime**: {e.g., TypeScript, Go, Python}
- **Framework**: {e.g., WXT, Next.js, Gin}
- **Key libraries**: {e.g., React, Fuse.js, xterm.js}
- **Testing**: {e.g., Vitest, Go test}
- **Build**: {e.g., Vite, esbuild}

## Repo Structure

```
{paste the relevant directory tree — focus on where workers will be writing}
```

## Coding Conventions

- {e.g., Conventional commits: feat:, fix:, chore:}
- {e.g., Each command implements the Command interface}
- {e.g., All state in chrome.storage — no module-level globals}
- {e.g., Shadow DOM: use px only, not rem}

## Key Specs

- Full spec: `{path/to/spec.md}`
- {Other relevant docs with paths}

## Known Gotchas

<!-- Orchestrator: populate BEFORE first worker dispatch with project-specific entries.
     Delete sections that don't apply to this project's stack. -->

<!-- Chrome Extensions (WXT/Plasmo):
- `Cannot find name 'chrome'` LSP diagnostics are false positives — build passes, ignore them
- Shadow DOM components must use `px` not `rem`
- Background service workers have no DOM access
-->

<!-- TypeScript monorepos:
- LSP may show errors for cross-package imports that resolve at build time
- `pnpm build` is ground truth, not LSP red squiggles
-->

<!-- Go modules:
- `go vet` false positives on generated code — check `//go:generate` before investigating
- Wire/inject errors often mean you need to run `go generate ./...` first
-->

<!-- Python:
- mypy errors on dynamic attrs (e.g., SQLAlchemy models) — if tests pass, ignore
- venv activation varies by OS — use `python -m` prefix for portability
-->

{Add project-specific gotchas here. Orchestrator appends more as workers discover recurring issues.}
