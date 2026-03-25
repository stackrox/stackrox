# 04 - Validation and Acceptance Spec

This document is the acceptance test contract for real-cluster validation.

## Preconditions

- `kubectl`, `curl`, `jq` installed.
- Logged into target cluster containing Compliance Operator resources.
- Central endpoint reachable from runner.
- Importer binary built locally.

Set environment:

```bash
export ROX_ENDPOINT="https://central.stackrox.example.com:443"
export ROX_API_TOKEN="<token>"
export ROX_ADMIN_USER="admin"
export ROX_ADMIN_PASSWORD="<password>"
export CO_NAMESPACE="openshift-compliance"
export IMPORTER_BIN="./bin/co-acs-scan-importer"
# For multi-cluster: merge kubeconfigs
export KUBECONFIG="~/.kube/config:~/.kube/config-secured-cluster"
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

- **IMP-ACC-002**: token and endpoint MUST pass read probe.
- **IMP-ACC-013**: optional basic-auth mode MUST pass read probe in local/dev environments.

Command:

```bash
curl -ksS \
  -H "Authorization: Bearer ${ROX_API_TOKEN}" \
  "${ROX_ENDPOINT}/v2/compliance/scan/configurations?pagination.limit=1" | jq .
```

Pass condition:

- command returns valid JSON and does not contain auth error.

Optional local/dev basic-auth probe:

```bash
curl -ksS \
  -u "${ROX_ADMIN_USER}:${ROX_ADMIN_PASSWORD}" \
  "${ROX_ENDPOINT}/v2/compliance/scan/configurations?pagination.limit=1" | jq .
```

### A3 - Dry-run side-effect safety

- **IMP-ACC-003**: dry-run MUST produce no writes.

Command (auto-discovery mode):

```bash
"${IMPORTER_BIN}" \
  --endpoint "${ROX_ENDPOINT}" \
  --dry-run \
  --report-json "/tmp/co-acs-import-dryrun.json"
```

Pass conditions:

- exit code is `0` or `2`,
- `/tmp/co-acs-import-dryrun.json` exists and is valid JSON,
- actions listed as planned only (no applied create/update markers),
- `problems[]` is present and contains `description` + `fixHint` for each problematic resource.

### A4 - Apply creates expected configs

- **IMP-ACC-004**: apply mode MUST create missing target ACS configs.

Command (auto-discovery mode):

```bash
"${IMPORTER_BIN}" \
  --endpoint "${ROX_ENDPOINT}" \
  --report-json "/tmp/co-acs-import-apply.json"
```

Verify:

```bash
curl -ksS \
  -H "Authorization: Bearer ${ROX_API_TOKEN}" \
  "${ROX_ENDPOINT}/v2/compliance/scan/configurations?pagination.limit=200" | \
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
  --endpoint "${ROX_ENDPOINT}" \
  --report-json "/tmp/co-acs-import-second-run.json"
```

Pass conditions:

- report shows skip actions for already-existing scan names,
- no net changes in ACS list output.

### A6 - Existing config behavior

- **IMP-ACC-006**: without `--overwrite-existing`, existing scan names MUST be skipped
  and recorded in `problems[]`.
- **IMP-ACC-014**: with `--overwrite-existing`, existing scan names MUST be updated via PUT.

Procedure (create-only):

1. Manually modify one imported ACS scan config (name unchanged).
2. Re-run importer without `--overwrite-existing`.
3. Verify that modified existing config is not updated and is captured as skipped conflict.

Procedure (overwrite):

1. Re-run importer with `--overwrite-existing`.
2. Verify that the modified config is updated back to the imported state.

### A8 - Multi-cluster merge

- **IMP-ACC-015**: when the same SSB name exists on multiple source clusters with matching
  profiles and schedule, importer MUST create one scan config targeting all resolved cluster IDs.
- **IMP-ACC-016**: when the same SSB name exists on multiple source clusters with different
  profiles or schedule, importer MUST error for that SSB name.

### A9 - Auto-discovery

- **IMP-ACC-017**: importer MUST auto-discover the ACS cluster ID from the admission-control
  ConfigMap's `cluster-id` key when no `--cluster` override is given.

### A7 - Failure paths

- **IMP-ACC-007**: invalid token MUST fail-fast with exit code `1`.
- **IMP-ACC-008**: missing referenced ScanSetting MUST fail only that binding (partial run exit code `2` when others succeed).
- **IMP-ACC-009**: transient ACS failures MUST follow retry policy and record attempt counts.
- **IMP-ACC-012**: all per-resource problems MUST be emitted in `problems[]` with remediation hint.

## Non-goal compliance checks

- **IMP-ACC-010**: no code changes in Sensor/Central runtime paths are required to run importer.
- **IMP-ACC-011**: importer MUST not mutate Compliance Operator resources.
