# 00 - Spec Process and Quality Gates

## Purpose

Translate product intent into executable behavior and contract specs before writing implementation code.

## Community best-practice principles applied

- **Behavior over implementation:** specs describe externally observable outcomes, not internal algorithms.
- **Single source of truth:** these specs replace ad-hoc task notes; code and tests must trace back to them.
- **Executable examples:** each important rule is captured as concrete scenario(s), preferably data-driven.
- **Contract-first boundaries:** external interfaces (CLI, ACS API payload shape, report output) are specified explicitly.
- **Low brittleness assertions:** tests assert fields that matter to consumers, avoid incidental details.

## Requirement key words

- `MUST`: mandatory behavior.
- `SHOULD`: strongly recommended unless justified deviation.
- `MAY`: optional.

## Traceability model

Every requirement gets an ID:
- `IMP-CLI-*` for CLI/config contract
- `IMP-MAP-*` for CO -> ACS mapping
- `IMP-IDEM-*` for idempotency/conflicts
- `IMP-ERR-*` for errors/retries/reporting
- `IMP-ACC-*` for acceptance/runtime checks

Implementation and tests MUST annotate requirement IDs in comments or test names.

## Spec execution strategy

### Unit-level specs
- Parsing/validation (flags, env, config file).
- Mapping translation (CO objects -> ACS payload).
- Diff/idempotency logic.
- Retry classification.

### Integration-level specs
- Kubernetes read path for CO resources.
- ACS API client interactions (`GET/POST/PUT`).
- Dry-run no-write guarantees.

### Acceptance-level specs
- End-to-end execution against real cluster and ACS endpoint.
- Idempotency second-run no-op behavior.

## Quality gates

Before merging implementation:

1. `MUST` requirements implemented.
2. All mapped scenarios have tests.
3. Dry-run validated as side-effect free.
4. Real-cluster acceptance checks pass.
5. No product runtime code path changes in Sensor/Central.
