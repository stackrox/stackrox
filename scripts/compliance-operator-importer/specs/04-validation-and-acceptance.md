# 04 - Validation and Acceptance Spec

This document is the acceptance test contract for real-cluster validation.

## Preconditions

- `kubectl`, `curl`, `jq` installed.
- Logged into target cluster containing Compliance Operator resources.
- ACS endpoint reachable from runner.
- Importer binary built locally.

Set environment:

```bash
export ACS_ENDPOINT="https://central.stackrox.example.com:443"
export ACS_API_TOKEN="<token>"
export ACS_USERNAME="<username>"
export ACS_PASSWORD="<password>"
export CO_NAMESPACE="openshift-compliance"
export IMPORTER_BIN="./bin/co-acs-scan-importer"
export ACS_CLUSTER_ID="<acs-cluster-id>"
```

## Acceptance checks

### A1 - CO resource discovery

- **IMP-ACC-001**: importer test run MUST begin only if required CO resource types are listable.

Commands:

```bash
kubectl get scansettingbindings.compliance.openshift.io -n "${CO_NAMESPACE}"
kubectl get scansettings.compliance.openshift.io -n "${CO_NAMESPACE}"
kubectl get profiles.compliance.openshift.io -n "${CO_NAMESPACE}"
kubectl get tailoredprofiles.compliance.openshift.io -n "${CO_NAMESPACE}" || true
```

Pass condition:

- first 3 commands succeed (exit 0).

### A2 - ACS auth preflight

- **IMP-ACC-002**: ACS token and endpoint MUST pass read probe.
- **IMP-ACC-013**: optional basic-auth mode MUST pass read probe in local/dev environments.

Command:

```bash
curl -ksS \
  -H "Authorization: Bearer ${ACS_API_TOKEN}" \
  "${ACS_ENDPOINT}/v2/compliance/scan/configurations?pagination.limit=1" | jq .
```

Pass condition:

- command returns valid JSON and does not contain auth error.

Optional local/dev basic-auth probe:

```bash
curl -ksS \
  -u "${ACS_USERNAME}:${ACS_PASSWORD}" \
  "${ACS_ENDPOINT}/v2/compliance/scan/configurations?pagination.limit=1" | jq .
```

### A3 - Dry-run side-effect safety

- **IMP-ACC-003**: dry-run MUST produce no writes.

Command:

```bash
"${IMPORTER_BIN}" \
  --acs-endpoint "${ACS_ENDPOINT}" \
  --acs-token-env ACS_API_TOKEN \
  --co-namespace "${CO_NAMESPACE}" \
  --acs-cluster-id "${ACS_CLUSTER_ID}" \
  --dry-run \
  --report-json "/tmp/co-acs-import-dryrun.json"
```

Pass conditions:

- exit code is `0` or `2`,
- `/tmp/co-acs-import-dryrun.json` exists and is valid JSON,
- actions listed as planned only (no applied create/update markers),
- `problems[]` is present and contains `description` + `fixHint` for each problematic resource.

### A4 - Apply creates expected configs (create-only)

- **IMP-ACC-004**: apply mode MUST create missing target ACS configs.

Command:

```bash
"${IMPORTER_BIN}" \
  --acs-endpoint "${ACS_ENDPOINT}" \
  --acs-token-env ACS_API_TOKEN \
  --co-namespace "${CO_NAMESPACE}" \
  --acs-cluster-id "${ACS_CLUSTER_ID}" \
  --report-json "/tmp/co-acs-import-apply.json"
```

Verify:

```bash
curl -ksS \
  -H "Authorization: Bearer ${ACS_API_TOKEN}" \
  "${ACS_ENDPOINT}/v2/compliance/scan/configurations?pagination.limit=200" | \
  jq '.configurations[] | {id, scanName, profiles: .scanConfig.profiles, description: .scanConfig.description}'
```

Pass conditions:

- expected imported scan names exist,
- profile lists match expected binding mappings.

### A5 - Idempotency on second run

- **IMP-ACC-005**: second run with same inputs MUST be no-op.

Command:

```bash
"${IMPORTER_BIN}" \
  --acs-endpoint "${ACS_ENDPOINT}" \
  --acs-token-env ACS_API_TOKEN \
  --co-namespace "${CO_NAMESPACE}" \
  --acs-cluster-id "${ACS_CLUSTER_ID}" \
  --report-json "/tmp/co-acs-import-second-run.json"
```

Pass conditions:

- report shows skip actions for already-existing scan names,
- no net changes in ACS list output.

### A6 - Existing config behavior (create-only)

- **IMP-ACC-006**: existing scan names MUST be skipped and recorded in `problems[]`.

Procedure:

1. Manually modify one imported ACS scan config (name unchanged).
2. Re-run importer.
3. Verify that modified existing config is not updated and is captured as skipped conflict.

### A7 - Failure paths

- **IMP-ACC-007**: invalid token MUST fail-fast with exit code `1`.
- **IMP-ACC-008**: missing referenced ScanSetting MUST fail only that binding (partial run exit code `2` when others succeed).
- **IMP-ACC-009**: transient ACS failures MUST follow retry policy and record attempt counts.
- **IMP-ACC-012**: all per-resource problems MUST be emitted in `problems[]` with remediation hint.

## Non-goal compliance checks

- **IMP-ACC-010**: no code changes in Sensor/Central runtime paths are required to run importer.
- **IMP-ACC-011**: importer MUST not mutate Compliance Operator resources.
