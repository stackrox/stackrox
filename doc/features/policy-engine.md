# Policy Engine

Core security policy evaluation across build, deploy, and runtime lifecycle phases with MITRE ATT&CK mapping and enforcement actions.

**Primary Packages**: `pkg/booleanpolicy`, `central/detection`, `central/policy`

## What It Does

Users create security policies with drag-and-drop criteria, receive real-time violation alerts, and configure enforcement actions (fail builds, block deployments, scale to zero). The system evaluates policies across build/deploy/runtime phases, tags policies with MITRE tactics/techniques, provides dry run mode for testing, and integrates with notification channels (Slack, PagerDuty, JIRA).

## Architecture

### Policy Evaluation Engine

The `pkg/booleanpolicy/` implements core compiler and evaluator. Augmented objects in `augmentedobjs/` enrich runtime data (deployments with image scans, network flows, processes). Field metadata in `fieldnames/` defines 200+ policy fields with type information. The evaluator in `evaluator/` implements policy matching logic.

### Detection Lifecycle

The `central/detection/` orchestrates policy evaluation across phases:

**Build Detection** (`detection/build/`): CI/CD pipeline checks, evaluates BUILD_TIME policies against images with CVE data enrichment.

**Deploy Detection** (`detection/deploy/`): Admission control and deployment-time checks, evaluates DEPLOY_TIME policies with image scan results, cluster context, runtime class, service accounts, volumes, and ports.

**Runtime Detection** (`detection/runtime/`): Process, network, and syscall evaluation, checks RUNTIME policies for unexpected processes, privilege escalation, baseline deviations.

**Lifecycle Manager** (`detection/lifecycle/`): Manages detection state transitions across phases.

**Alert Generation** (`central/alert/`): Creates alerts from policy violations linking deployment, policy, and indicators.

### Policy Data Model

**Policy** proto: Metadata (ID, name, description, severity, categories), policy sections (grouped criteria with OR logic between sections), policy groups (criteria within section with AND logic), lifecycle stages, enforcement actions, scope (cluster/namespace/labels), MITRE mapping, exclusions (deployment/image patterns), and disabled state.

**PolicyCriteria**: Field name (standardized identifier like "Privileged Container" or "CVE Severity"), operator (EQUALS, NOT_EQUALS, GREATER_THAN, REGEX_MATCH), values list, and negated flag.

**Alert**: Policy reference, violated deployment/resource, lifecycle stage, violation messages for failed criteria, state (ACTIVE/RESOLVED/ATTEMPTED), enforcement action taken, and first/last occurrence timestamps.

### Augmented Objects

The engine evaluates against objects combining runtime state with enrichment:

- **Deployment**: K8s spec + image scans + network flows + runtime processes
- **Pod**: Container specs + volumes + security context
- **Image**: Metadata + CVEs + components + signature verification
- **Process**: Process tree + args + user/group + file access
- **NetworkFlow**: Source/destination + ports + protocols + baselines

Field metadata defines all available policy fields with types (string, bool, enum, numeric, string array).

## Data Flow

### Build-Time Detection

1. **Request**: `roxctl image check` sends image metadata + scan results to Central
2. **Evaluation**: `central/detection/build/` evaluates all BUILD_TIME policies against enriched image (CVE data, components, Dockerfile analysis)
3. **Policy Matching**: Each section evaluated independently (OR), groups within section together (AND), first matching section triggers violation
4. **Response**: Return pass/fail + violation details, exit with error code if enforcement enabled

### Deploy-Time Detection

1. **Admission Webhook**: Kubernetes calls Sensor on deployment create/update, Sensor forwards spec to Central via gRPC
2. **Enrichment**: Deployment augmented with image scan results, cluster context (namespace/labels), runtime class/service accounts, volume types/port configs
3. **Enforcement**: If policy violated with FAIL_BUILD_ENFORCEMENT: deny admission; with SCALE_TO_ZERO: allow but scale replicas to 0; without enforcement: allow and create alert
4. **Alert**: `central/alert/datastore/` creates record linking deployment and policy, sends notifications via integrations

### Runtime Detection

1. **Event Collection**: Collector sends process/network/file events to Sensor, which aggregates and forwards ProcessIndicator and NetworkFlow messages
2. **Evaluation**: `central/detection/runtime/` receives indicators, augments with deployment context and baselines, evaluates RUNTIME policies (unexpected process, baseline deviation)
3. **Baseline Learning**: Process baselines auto-lock after observation, network baselines track expected connections, policies enforce deviations
4. **Alert**: Runtime violations create alerts, enforcement limited (cannot block running processes), network policy generation can isolate pods

### Policy Set Management

The `central/policySet/` manages default and custom collections. Migration includes updated default policies in new versions. Reconciliation preserves user modifications during upgrades. Deduplication prevents duplicate policy IDs.

## Configuration

**Central**:
- `ROX_POLICIES_DIR`: Default policy definitions directory (default: /policies)
- `ROX_POLICY_EXCLUSION_ENABLED`: Enable exclusions feature
- `ROX_PROCESS_EXCLUSION_ENABLED`: Enable process-level exclusions

**Sensor**:
- `ROX_ADMISSION_CONTROL_LISTEN_ON_CREATES`: Enable webhook on creates (default: false)
- `ROX_ADMISSION_CONTROL_LISTEN_ON_UPDATES`: Enable webhook on updates (default: true)
- `ROX_ADMISSION_CONTROL_LISTEN_ON_EVENTS`: Enable webhook on events (default: false)
- `ROX_ENABLE_RUNTIME_POLICIES`: Enable runtime evaluation (default: true)

Admission control per-cluster settings: cluster-wide enforcement enable/disable, webhook timeout in seconds (default: 3), contact Central timeout grace period, enforcement behavior (fail open vs closed), scan on admission trigger.

Policy configuration options: enforcement actions per policy (none/inform/enforce), lifecycle stage selection, enforcement behavior (scale to zero/block/fail build), scope (cluster inclusion/exclusion, namespace labels, image patterns), exclusions (deployment name/scope, image name/tag/registry).

## Testing

**Unit Tests**:
- `pkg/booleanpolicy/*_test.go`: Compilation and evaluation
- `pkg/booleanpolicy/evaluator/*_test.go`: Criterion matching
- `central/detection/build/*_test.go`: Build-time detection
- `central/detection/deploy/*_test.go`: Deploy-time with mocks
- `central/detection/runtime/*_test.go`: Runtime indicators

**E2E**: `PolicyEnforcementTest.groovy`, `AdmissionControllerTest.groovy`, `NetworkPolicyTest.groovy`, `RuntimePolicyTest.groovy` in `qa-tests-backend/`

## Known Limitations

**Performance**: 1000+ deployment policy evaluation takes seconds. Many nested criteria slow evaluation. High-cardinality runtime policies generate excessive alerts.

**Behavior**: Deployment creation before webhook ready causes race. Process baselines become stale in dynamic environments. Multiple policies with different enforcement actions conflict.

**Features**: Not all Kubernetes fields available in policies. Cannot kill processes or block syscalls at runtime, only alert. Complex regex causes timeouts. No guaranteed policy evaluation ordering or dependencies.

**Workarounds**: Use policy exclusions instead of deletion. Scope policies to specific namespaces to reduce overhead. Regularly review and prune runtime baselines. Test policies in dry-run mode before enforcement. Use `ROX_ADMISSION_CONTROL_TIMEOUT` to increase webhook timeout.

## Implementation

**Engine**: `pkg/booleanpolicy/policyversion/`, `pkg/booleanpolicy/evaluator/`, `pkg/booleanpolicy/augmentedobjs/`
**Detection**: `central/detection/build/`, `central/detection/deploy/`, `central/detection/runtime/`, `central/detection/lifecycle/`
**Management**: `central/policy/datastore/`, `central/policy/service/`, `central/policy/matcher/`
**Enforcement**: `sensor/admission-control/`, `central/networkpolicies/`, `sensor/common/enforcement/`
**API**: `proto/api/v1/policy_service.proto`, `proto/storage/policy.proto`, `proto/storage/alert.proto`
