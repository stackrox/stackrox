# Compliance Operator -> ACS Importer (Spec Set)

This directory contains **specifications only** for a standalone importer that reads existing Compliance Operator resources and creates equivalent ACS compliance scan configurations via ACS API.

No runtime changes to Sensor/Central are in scope for this work item.
Phase 1 mode is **create-only** (no ACS updates).

## Spec-driven workflow

Implement in this order:

1. Read `DECISIONS.md` (frozen v1 scope and non-goals).
2. Read `specs/00-spec-process.md` (process and quality gates).
3. Use `specs/06-implementation-backlog.md` to execute slice-by-slice.
4. Implement CLI contract from `specs/01-cli-and-config-contract.md`.
5. Implement behavior scenarios in:
   - `specs/02-co-to-acs-mapping.feature`
   - `specs/03-idempotency-dry-run-retries.feature`
6. Validate with `specs/04-validation-and-acceptance.md`.

Definition of done:

- every MUST statement in spec docs is implemented,
- every `Scenario` in `.feature` files has an automated test,
- resource-level issues are skipped and captured in `problems[]` with fix hints,
- acceptance commands in `specs/04-validation-and-acceptance.md` pass on a real cluster.
