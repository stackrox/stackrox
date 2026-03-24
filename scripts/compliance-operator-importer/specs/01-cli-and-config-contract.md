# 01 - CLI and Config Contract

## Goal

Define the importer interface so it can be implemented and tested predictably.

## Inputs contract

### Required inputs

- **IMP-CLI-001**: importer MUST accept ACS endpoint (`--acs-endpoint` or `ACS_ENDPOINT`).
- **IMP-CLI-002**: importer MUST support ACS auth modes:
  - token mode (default): bearer token from env var (`--acs-token-env`, default `ACS_API_TOKEN`),
  - basic mode (optional): username/password.
- **IMP-CLI-003**: importer MUST support source cluster selection like `kubectl`:
  - by default, use current kube context,
  - optionally accept `--source-kubecontext <name>` to pick a specific context.
- **IMP-CLI-004**: importer MUST support namespace scope:
  - `--co-namespace <ns>` for single namespace, or
  - `--co-all-namespaces` for cluster-wide scan.
- **IMP-CLI-005**: importer MUST accept one destination ACS cluster ID:
  - `--acs-cluster-id <id>`.
  - all imported scan configs target this ACS cluster ID.

### Optional inputs

- **IMP-CLI-006**: importer mode is create-only for phase 1.
- **IMP-CLI-007**: `--dry-run` MUST disable all ACS write operations.
- **IMP-CLI-008**: `--report-json <path>` for structured report output.
- **IMP-CLI-009**: `--request-timeout <duration>` default `30s`.
- **IMP-CLI-010**: `--max-retries <int>` default `5`, min `0`.
- **IMP-CLI-011**: `--ca-cert-file <path>` optional.
- **IMP-CLI-012**: `--insecure-skip-verify` default false; MUST require explicit flag.
- **IMP-CLI-023**: importer MUST accept `--acs-auth-mode` enum:
  - `token` (default)
  - `basic`
- **IMP-CLI-024**: for basic mode, importer MUST accept:
  - `--acs-username` or `ACS_USERNAME`
  - `--acs-password-env` (default `ACS_PASSWORD`) to read password from env var.
- **IMP-CLI-025**: importer MUST reject ambiguous auth config (for example, missing required values for chosen mode).

## Preflight checks

- **IMP-CLI-013**: endpoint MUST be `https://`.
- **IMP-CLI-014**: auth material for selected mode MUST be non-empty:
  - token mode: resolved token is non-empty,
  - basic mode: username and password are non-empty.
- **IMP-CLI-015**: importer MUST probe ACS auth with:
  - `GET /v2/compliance/scan/configurations?pagination.limit=1`
  - using selected auth mode,
  - success only on HTTP 200.
- **IMP-CLI-016**: HTTP 401/403 at preflight MUST fail-fast with remediation message.
- **IMP-CLI-026**: when auth mode is not explicitly set, importer MUST default to `token`.

## Output contract

### Exit codes

- **IMP-CLI-017**: `0` when run completed with no failed bindings.
- **IMP-CLI-018**: `1` for fatal preflight/config errors (no import attempted).
- **IMP-CLI-019**: `2` for partial success (some bindings failed).

### Console summary

- **IMP-CLI-020**: print totals:
  - bindings discovered
  - creates/skips/failures
  - dry-run indicator

### JSON report shape

- **IMP-CLI-021**: when `--report-json` is set, write valid JSON with:
  - `meta` (timestamp, dryRun, namespaceScope, mode=`create-only`)
  - `counts` (discovered, create, skip, failed)
  - `items[]`:
    - `source` (`namespace`, `bindingName`, `scanSettingName`)
    - `action` (`create|skip|fail`)
    - `reason`
    - `attempts`
    - `acsScanConfigId` (if known)
    - `error` (if failed)
  - `problems[]`:
    - `severity` (`error|warning`)
    - `category` (`input|mapping|conflict|auth|api|retry|validation`)
    - `resourceRef` (`namespace/name` or synthetic ref for non-resource errors)
    - `description` (what happened)
    - `fixHint` (how to fix)
    - `skipped` (boolean; true when resource was skipped)

- **IMP-CLI-022**: whenever any problem occurs for a resource, importer MUST:
  - skip that resource,
  - append one `problems[]` entry with `description` and `fixHint`,
  - continue processing other resources.

## Existing ACS config behavior (create-only)

- **IMP-IDEM-001**: if `scanName` already exists in ACS, importer MUST skip that source resource.
- **IMP-IDEM-002**: skipped-existing resources MUST be added to `problems[]` with category `conflict` and a fix hint.
- **IMP-IDEM-003**: create-only phase MUST NOT send `PUT` updates.

Example minimal report skeleton:

```json
{
  "meta": {
    "dryRun": true,
    "namespaceScope": "openshift-compliance",
    "mode": "create-only"
  },
  "counts": {
    "discovered": 2,
    "create": 1,
    "skip": 1,
    "failed": 0
  },
  "items": [],
  "problems": []
}
```
