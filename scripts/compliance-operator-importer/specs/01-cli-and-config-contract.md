# 01 - CLI and Config Contract

## Goal

Define the importer interface so it can be implemented and tested predictably.

## Inputs contract

### Required inputs

Note: flag names and environment variables are aligned with `roxctl` conventions.

- **IMP-CLI-001**: importer MUST accept Central endpoint (`--endpoint` or `ROX_ENDPOINT`).
  - if the value does not contain a scheme, importer MUST prepend `https://`.
  - if the value starts with `http://`, importer MUST error.
- **IMP-CLI-002**: importer MUST support auth modes, auto-inferred from available credentials
  (aligned with `roxctl` behavior — no explicit `--auth-mode` flag, no env-var-name indirection):
  - token mode: when `ROX_API_TOKEN` is set,
  - basic mode: when `ROX_ADMIN_PASSWORD` is set,
  - if both are set: error ("ambiguous auth"),
  - if neither is set: error with help text listing both options.
- **IMP-CLI-003**: importer MUST support multi-cluster source selection via two mechanisms:
  - `--kubeconfig <path>` (repeatable): each path is a separate source cluster, using that file's
    current context. This is the primary mechanism for users with one kubeconfig file per cluster.
  - `--kubecontext <name>` (repeatable): selects contexts within the active kubeconfig
    (set via `KUBECONFIG` env var or `~/.kube/config`). Use when a single merged kubeconfig
    contains unique context names for all clusters.
  - `--kubecontext all`: iterates all contexts in the active kubeconfig.
  - `--kubeconfig` and `--kubecontext` are mutually exclusive (error if both given).
  - when neither `--kubeconfig` nor `--kubecontext` is given, importer MUST use
    the current kubeconfig context (single-cluster mode, backward compatible).
  - help text MUST suggest:
    - using `--kubeconfig` (repeatable) when clusters have separate kubeconfig files, or
    - merging kubeconfigs (`KUBECONFIG=a.yaml:b.yaml`) with unique context names
      and using `--kubecontext`.
- **IMP-CLI-004**: importer MUST support namespace scope:
  - `--co-namespace <ns>` (default `openshift-compliance`) for single namespace, or
  - `--co-all-namespaces` for cluster-wide scan.
- **IMP-CLI-005**: importer MUST support ACS cluster identification via `--cluster`:
  - by default (no `--cluster` flag), auto-discover the ACS cluster ID for each source
    cluster (see IMP-MAP-016..018).
  - `--cluster <value>` accepts three forms:
    - UUID: used directly as the ACS cluster ID (single-cluster shorthand).
    - name: resolved to an ACS cluster ID via `GET /v1/clusters` (single-cluster shorthand).
    - `<kubecontext>=<name-or-uuid>` (repeatable): maps a specific kubeconfig context to
      an ACS cluster, overriding auto-discovery for that context.

### Optional inputs

- **IMP-CLI-006**: importer default mode is create-only; `--overwrite-existing` enables update mode.
- **IMP-CLI-007**: `--dry-run` MUST disable all ACS write operations.
- **IMP-CLI-008**: `--report-json <path>` for structured report output.
- **IMP-CLI-009**: `--request-timeout <duration>` default `30s`.
- **IMP-CLI-010**: `--max-retries <int>` default `5`, min `0`.
- **IMP-CLI-011**: `--ca-cert-file <path>` optional.
- **IMP-CLI-012**: `--insecure-skip-verify` default false; MUST require explicit flag.
- **IMP-CLI-023**: (removed — auth mode is auto-inferred, see IMP-CLI-002).
- **IMP-CLI-024**: for basic mode:
  - username is read from `--username` flag or `ROX_ADMIN_USER` env var (default `admin`).
  - password is read from `ROX_ADMIN_PASSWORD` env var (no flag; aligned with roxctl).
- **IMP-CLI-025**: importer MUST reject ambiguous auth config:
  - both `ROX_API_TOKEN` and `ROX_ADMIN_PASSWORD` are set → error,
  - neither is set → error with help text.
- **IMP-CLI-027**: `--overwrite-existing` (default `false`):
  - when `false`: existing ACS scan configs with matching `scanName` are skipped (create-only).
  - when `true`: existing ACS scan configs with matching `scanName` are updated via
    `PUT /v2/compliance/scan/configurations/{id}`.

## Preflight checks

- **IMP-CLI-013**: `--endpoint` MUST use HTTPS:
  - bare hostname/port (no scheme) → `https://` is prepended automatically,
  - `https://...` → accepted as-is,
  - `http://...` → error.
- **IMP-CLI-014**: auth material for inferred mode MUST be non-empty:
  - token mode: `ROX_API_TOKEN` is non-empty,
  - basic mode: `ROX_ADMIN_PASSWORD` is non-empty (username defaults to `admin`).
- **IMP-CLI-015**: importer MUST probe ACS auth with:
  - `GET /v2/compliance/scan/configurations?pagination.limit=1`
  - using selected auth mode,
  - success only on HTTP 200.
- **IMP-CLI-016**: HTTP 401/403 at preflight MUST fail-fast with remediation message.
- **IMP-CLI-026**: (removed — auth mode is auto-inferred, see IMP-CLI-002).

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
  - `meta` (timestamp, dryRun, namespaceScope, mode=`create-only` | `create-or-update`)
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

## Existing ACS config behavior

- **IMP-IDEM-001**: when `--overwrite-existing` is `false` (default) and `scanName` already exists
  in ACS, importer MUST skip that source resource.
- **IMP-IDEM-002**: skipped-existing resources MUST be added to `problems[]` with category `conflict`
  and a fix hint.
- **IMP-IDEM-003**: when `--overwrite-existing` is `false`, importer MUST NOT send `PUT` updates.
- **IMP-IDEM-008**: when `--overwrite-existing` is `true` and `scanName` already exists in ACS,
  importer MUST update it via `PUT /v2/compliance/scan/configurations/{id}`.
- **IMP-IDEM-009**: when `--overwrite-existing` is `true` and `scanName` does not exist,
  importer MUST create it via `POST` (same as IMP-IDEM-001 create path).

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
