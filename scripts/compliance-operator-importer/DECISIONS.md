# V1 Scope Freeze: CO -> ACS Importer

## Status

This document freezes Phase 1 behavior. Any deviation requires updating this file and corresponding specs.

## Frozen decisions

1. **Execution model**
   - Standalone external importer only.
   - No runtime/product code changes in Sensor/Central/ACS backend.

2. **Importer mode**
   - Phase 1 is create-only.
   - Importer may create new ACS scan configs.
   - Importer must never update existing ACS scan configs.

3. **Implementation language**
   - Use **Go** for Phase 1 implementation.
   - Do not implement Phase 1 importer in bash/shell.
   - Python is an acceptable future alternative only if explicitly re-decided in this file.

4. **Existing-name behavior**
   - If `scanName` already exists in ACS, skip resource.
   - Add one entry to `problems[]` with clear `description` and `fixHint`.

5. **Error handling model**
   - Resource-level issue => skip resource, continue processing, emit `problems[]` entry.
   - Fatal preflight/config issue => fail run before resource processing.

6. **Cluster targeting**
   - Source cluster selected like `kubectl` (current context by default, optional context override).
   - Single destination ACS cluster ID via `--acs-cluster-id`.

7. **ACS authentication model**
   - Default auth mode is token (`ACS_API_TOKEN` via `--acs-token-env`).
   - Optional basic-auth mode is allowed for local/dev environments.
   - Basic mode uses username/password inputs and the same preflight endpoint checks.

8. **Profile kind fallback**
   - Missing `ScanSettingBinding.profiles[].kind` defaults to `Profile` (profiles is a top-level field, not under spec).

9. **Schedule conversion**
   - Convert valid CO cron to ACS schedule fields.
   - Conversion failure => skip resource + `problems[]` entry with remediation hint.

10. **Provenance marker**

- Not required in Phase 1 create-only mode.
- Can be revisited in a future update/reconcile phase.

## Deferred to Phase 2 (out of scope)

- Update/reconcile mode (`PUT`) for existing configs.
- Ownership/provenance-based update guard.
- Multi-target cluster mapping per binding.
