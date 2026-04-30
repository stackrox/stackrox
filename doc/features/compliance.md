# Compliance

Real-time compliance assessment across security standards using built-in checks (V1) and OpenShift Compliance Operator integration (V2).

**Primary Packages**: `pkg/compliance`, `central/compliance`, `central/complianceoperator`

## What It Does

StackRox provides two compliance systems: legacy built-in checks for deployments/nodes/clusters, and integration with OpenShift Compliance Operator for comprehensive CIS/STIG profiles. Users see compliance scores, passing/failing checks with evidence, scan scheduling, historical trends, and remediation guidance.

Supported standards include PCI DSS, HIPAA, NIST 800-190, NIST SP 800-53, CIS Kubernetes Benchmark, and various STIG profiles.

## Architecture

### Legacy Compliance (V1)

The `pkg/compliance/` framework executes checks against cluster resources. Check implementations in `pkg/compliance/checks/` cover 100+ security controls. The `central/compliance/manager/` orchestrates scans and scheduling while `central/compliance/datastore/` persists results.

Standard definitions in `pkg/compliance/framework/` map checks to compliance frameworks. Each ComplianceRunResults proto tracks domain (deployment/node/cluster), standard, check results list, cluster ID, and completion timestamp.

ComplianceCheckResult contains check ID reference, state (PASS/FAIL/ERROR/INFO/SKIP), evidence as JSON, and target resource identifier.

### Compliance Operator (V2)

The `central/complianceoperator/v2/` architecture manages profile/scan configuration/result lifecycles. Key modules include:

- `profiles/`: Profile and rule storage
- `scanconfigurations/`: Scan config and settings bindings
- `checkresults/`: Result processing pipeline
- `report/`: Compliance report generation
- `integration/`: ComplianceIntegration CR management

ComplianceScanConfiguration defines scan name, profile names, target clusters, schedule (cron or one-time), and strict node scan settings.

ComplianceProfile specifies product type (node/platform/OCP4), standard, description, rule list, and TailoredProfile variable assignments.

ComplianceCheckResult includes check ID, status (PASS/FAIL/MANUAL/INFO/INCONSISTENT/ERROR/NOT_APPLICABLE), severity, evidence as Valueslist, and rationale.

## Data Flow

### Legacy Scan (V1)

1. **Initiation**: User triggers via UI/API, manager creates job targeting specific cluster and standard
2. **Execution**: Load check registry, filter by standard and domain, execute against target data (deployments from datastore, nodes from sensor), collect evidence
3. **Aggregation**: Results aggregate by standard/domain, calculate overall score (% passing), store in datastore
4. **Reporting**: Results available via API/UI, export to CSV/PDF via `central/reports/`

Examples: PCI DSS 1.1.4 checks for NetworkPolicies, HIPAA 164.312(a)(1) validates pod security contexts, NIST 800-190 confirms image scanning.

### Compliance Operator Scan (V2)

1. **Profile Discovery**: Compliance Operator publishes Profile CRs in OpenShift, Sensor watches and syncs to Central, profiles store in `v2/profiles/`
2. **Configuration**: User creates ComplianceScanConfiguration selecting profiles, clusters, schedule
3. **Integration Creation**: Central creates ComplianceIntegration CR in target clusters via `v2/integration/`
4. **Execution**: Compliance Operator reads CR, creates ScanSettingBinding, launches scan pods, runs OpenSCAP against profiles, writes ComplianceCheckResult CRs
5. **Synchronization**: Sensor watches results, syncs to Central via gRPC, `v2/checkresults/` processes and links to configs
6. **Reporting**: `v2/report/` aggregates pass/fail counts, provides remediation instructions, enables CSV export

## Configuration

**Central Environment**:
- `ROX_COMPLIANCE_ENABLED`: Enable V1 (default: true)
- `ROX_COMPLIANCE_OPERATOR_INTEGRATION`: Enable V2 (default: true)
- `ROX_COMPLIANCE_OPERATOR_AUTO_DISCOVERY`: Auto-discover profiles (default: true)

**Sensor**: `ROX_COMPLIANCE_OPERATOR_ENABLED`: Enable operator watching (default: true)

**V1 Settings** (via API): Standard selection, node scanning enable/disable, periodic scan interval.

**V2 Settings**: Profile selection from discovered options, cluster selection, cron schedule (e.g., "0 2 * * *"), StrictNodeScan requirement, suspend flag, TailoredProfile variable overrides.

V2 requires OpenShift Compliance Operator installation in cluster, which auto-creates Profile CRs for Central discovery.

## Testing

**Unit Tests**:
- `pkg/compliance/checks/*_test.go`: Individual check logic
- `pkg/compliance/framework/*_test.go`: Standard definitions
- `central/compliance/manager/*_test.go`: Scan orchestration
- `central/complianceoperator/v2/*/datastore/*_test.go`: CRUD operations
- `central/complianceoperator/v2/checkresults/*_test.go`: Result processing

**Integration Tests** (PostgreSQL, `//go:build sql_integration`):
- `central/complianceoperator/v2/*/datastore/internal/store/postgres/*_test.go`
- Requires PostgreSQL on port 5432

**E2E**: `ComplianceTest.groovy` (V1), `ComplianceOperatorTest.groovy` (V2) in `qa-tests-backend/`

## Known Limitations

**Performance**: V1 scans of large clusters (1000+ deployments) take 10+ minutes. OpenSCAP scans create CPU spikes. Result storage grows unbounded.

**Accuracy**: V1 has limited host filesystem visibility. Some checks produce false positives. Not all V2 profiles apply to all cluster types.

**Behavior**: V1 scheduled scans rarely used; most run on-demand. V2 profile discovery delays after Operator install. Multiple overlapping scans can interfere.

**Extensibility**: V1 cannot add custom checks without code changes. V2 requires OpenShift Compliance Operator, not generic Kubernetes. No fleet-wide result aggregation. No auto-remediation.

**Workarounds**: Use V2 for comprehensive host-level compliance (CIS/STIG), V1 for quick deployment checks. Export to external GRC tools for retention. Increase Operator scan timeout for large nodes. Filter standards to reduce V1 scan time. Run scans during low-traffic windows.

## Implementation

**V1 Check Execution**:
- Check registry: `pkg/compliance/checks/standards/standard.go` (RegisterChecksForStandard)
- Check implementations: `pkg/compliance/checks/kubernetes/`, `pkg/compliance/checks/pcidss32/`, `pkg/compliance/checks/hipaa_164/`
- Scan orchestration: `central/compliance/manager/manager_impl.go` (TriggerRuns, ProcessRun)
- Check execution: `central/compliance/manager/checks.go` (runChecksForStandard, runCheck)
- Result collection: `central/compliance/manager/collect_results.go` (collectStandardResults)
- Scheduling: `central/compliance/manager/run.go` (scheduleRun)

**V1 Infrastructure**: `pkg/compliance/framework/`, `central/compliance/manager/`, `central/compliance/datastore/`

**V2 Execution**:
- Profile discovery: `central/complianceoperator/v2/profiles/datastore/datastore_impl.go`
- Scan configuration: `central/complianceoperator/v2/scanconfigurations/manager/manager.go`
- Result processing: `central/complianceoperator/v2/checkresults/datastore/datastore_impl.go`
- Integration CR creation: `central/complianceoperator/v2/integration/manager/manager.go`

**V2 Infrastructure**: `central/complianceoperator/v2/profiles/`, `v2/scanconfigurations/`, `v2/checkresults/`, `v2/report/`, `v2/integration/`

**API**: `proto/api/v1/compliance_service.proto`, `proto/api/v2/compliance_profile_service.proto`
**Storage**: `proto/storage/compliance.proto`, `proto/storage/compliance_operator.proto`
