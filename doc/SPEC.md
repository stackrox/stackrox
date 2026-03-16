# StackRox / RHACS System Specification

**Version:** 1.0
**Status:** Living Document
**Scope:** Defines the behavior, contracts, data model, and pipelines of the StackRox/RHACS Kubernetes security platform precisely enough that a compatible implementation could be built from this document alone.

---

## Table of Contents

1. [Problem Statement](#1-problem-statement)
2. [Goals and Non-Goals](#2-goals-and-non-goals)
3. [System Overview](#3-system-overview)
4. [Core Domain Model](#4-core-domain-model)
5. [Communication Protocol](#5-communication-protocol)
6. [Data Pipelines](#6-data-pipelines)
7. [Authorization Model (SAC)](#7-authorization-model-sac)
8. [Policy Engine Specification](#8-policy-engine-specification)
9. [Storage Model](#9-storage-model)
10. [Search Specification](#10-search-specification)
11. [Scanner V4 Specification](#11-scanner-v4-specification)
12. [Collector Specification](#12-collector-specification)
13. [Risk Scoring Model](#13-risk-scoring-model)
14. [Notification and Enforcement](#14-notification-and-enforcement)
15. [Compliance Model](#15-compliance-model)
16. [Certificate and Identity Model](#16-certificate-and-identity-model)
17. [Failure Model and Recovery](#17-failure-model-and-recovery)
18. [Configuration Reference](#18-configuration-reference)

---

## 1. Problem Statement

### What StackRox IS

StackRox (Red Hat Advanced Cluster Security, RHACS) is a Kubernetes-native security platform that provides:

- Continuous visibility into the security posture of Kubernetes clusters
- Automated detection and response for threats across the software lifecycle (build, deploy, runtime)
- Vulnerability management for container images and cluster nodes
- Network traffic analysis and anomaly detection for Kubernetes workloads
- Compliance assessment against industry standards
- Policy-driven enforcement at admission time, deploy time, and runtime

### What StackRox IS NOT

- **Not a SIEM.** StackRox generates alerts and forwards them to external systems (Splunk, Elasticsearch, etc.) but does not provide log aggregation, long-term event storage, or cross-platform correlation.
- **Not a network firewall or WAF.** StackRox observes and visualizes network flows and evaluates them against baselines. It does not inspect packet payloads, perform DPI, or block traffic inline. Enforcement is achieved through Kubernetes NetworkPolicy generation, not inline filtering.
- **Not a general-purpose container orchestrator.** StackRox depends on an existing Kubernetes cluster. It monitors and secures workloads but does not schedule or manage them.
- **Not a CI/CD system.** StackRox integrates with CI/CD pipelines via `roxctl` CLI for build-time image scanning and policy checks but does not orchestrate builds or deployments.
- **Not a secrets manager.** StackRox inventories Kubernetes Secrets for security posture visibility but does not provide secrets storage, rotation, or injection.

### Important Boundaries

1. **Single Central, Multiple Clusters.** One Central instance manages security for N Kubernetes clusters. Each cluster runs exactly one Sensor.
2. **Kubernetes-only.** The system targets Kubernetes and OpenShift. Non-Kubernetes container runtimes are not supported as first-class citizens.
3. **PostgreSQL-only storage.** All persistent state resides in PostgreSQL. There is no pluggable storage backend.
4. **Go-only backend.** All server-side components are implemented in Go. The UI is React/TypeScript.
5. **gRPC-primary communication.** Inter-component communication uses gRPC with mTLS. External APIs are exposed as gRPC + gRPC-Gateway (REST/JSON).

---

## 2. Goals and Non-Goals

### Goals

| ID | Goal | Mechanism |
|----|------|-----------|
| G1 | **Vulnerability Management** | Scanner V4 indexes container images, matches packages against CVE databases, enriches with CVSS/EPSS, and presents prioritized vulnerability data. |
| G2 | **Runtime Threat Detection** | Collector captures process executions, network connections, and file access via eBPF. Sensor evaluates these against runtime policies and baselines. |
| G3 | **Policy Enforcement** | Policies are evaluated at build time (roxctl), deploy time (admission webhook), and runtime (process/network/file events). Enforcement actions include fail-build, block-admission, scale-to-zero, and kill-pod. |
| G4 | **Network Visibility** | eBPF-based network flow collection, delta-based aggregation, baseline learning, and graph visualization of inter-deployment traffic. |
| G5 | **Compliance Assessment** | Integration with Compliance Operator for CIS, PCI-DSS, HIPAA, NIST standards. Result aggregation and reporting. |
| G6 | **Risk Prioritization** | Multiplicative risk scoring model combining CVE severity, policy violations, network exposure, RBAC permissions, process baseline deviations, and image age. |
| G7 | **Multi-Cluster Management** | Single Central manages security across multiple Kubernetes/OpenShift clusters with per-cluster Sensor agents. |
| G8 | **Scoped Access Control** | Fine-grained RBAC with three-state logic (Included/Partial/Excluded) across cluster and namespace hierarchies. |

### Non-Goals

| ID | Non-Goal | Rationale |
|----|----------|-----------|
| NG1 | Log aggregation and SIEM functionality | Alerts are forwarded to external systems. |
| NG2 | Inline network traffic filtering | Observation-only; enforcement via K8s NetworkPolicy. |
| NG3 | Host-level IDS/IPS for non-containerized workloads | Scope limited to Kubernetes workloads. |
| NG4 | Supply chain attestation and signing | StackRox verifies Cosign signatures but does not generate them. |
| NG5 | Application-level vulnerability scanning (DAST/SAST) | Only OS and language package CVE scanning. |
| NG6 | Multi-tenancy within a single Central | One Central serves one security team. RBAC provides namespace-level scoping, not full tenant isolation. |

---

## 3. System Overview

### Component Architecture

```
                    ┌─────────────────────────────────────────────┐
                    │            Management Cluster                │
                    │                                              │
                    │  ┌──────────┐   ┌──────────┐   ┌─────────┐ │
                    │  │ Central  │◄──│ Scanner  │   │ Scanner │ │
                    │  │ (API +   │   │   V4     │   │   DB    │ │
                    │  │  Engine) │   │(Indexer+ │   │(Postgres│ │
                    │  │          │   │ Matcher) │   │  15+)   │ │
                    │  └────┬─────┘   └──────────┘   └─────────┘ │
                    │       │                                      │
                    │  ┌────┴─────┐                                │
                    │  │Central DB│                                │
                    │  │(Postgres │                                │
                    │  │  15+)    │                                │
                    │  └──────────┘                                │
                    └───────┬─────────────────────────────────────┘
                            │ gRPC (mTLS)
              ┌─────────────┼─────────────────┐
              │             │                 │
    ┌─────────┴──┐  ┌───────┴────┐  ┌─────────┴──┐
    │ Cluster A  │  │ Cluster B  │  │ Cluster C  │
    │            │  │            │  │            │
    │ ┌────────┐ │  │ ┌────────┐ │  │ ┌────────┐ │
    │ │ Sensor │ │  │ │ Sensor │ │  │ │ Sensor │ │
    │ └───┬────┘ │  │ └────────┘ │  │ └────────┘ │
    │     │      │  │            │  │            │
    │ ┌───┴────┐ │  │            │  │            │
    │ │Admissn.│ │  │            │  │            │
    │ │Control │ │  │            │  │            │
    │ └────────┘ │  │            │  │            │
    │            │  │            │  │            │
    │ ┌────────┐ │  │            │  │            │
    │ │Collectr│ │  │            │  │            │
    │ │(DaemonS│ │  │            │  │            │
    │ │  et)   │ │  │            │  │            │
    │ └────────┘ │  │            │  │            │
    └────────────┘  └────────────┘  └────────────┘
```

### Component Responsibilities

#### Central

| Aspect | Detail |
|--------|--------|
| **Type** | Stateful service (Deployment, 1 replica) |
| **Language** | Go |
| **Database** | PostgreSQL 15+ |
| **APIs** | gRPC + REST (gRPC-Gateway), 60+ v1 services, 10+ v2 services |
| **Responsibilities** | API server, policy engine, alert management, image enrichment, risk scoring, vulnerability management, compliance aggregation, user authentication/authorization, notification routing, cluster management |
| **Ports** | 8443 (gRPC/HTTPS) |

#### Sensor

| Aspect | Detail |
|--------|--------|
| **Type** | Stateless service (Deployment, 1 replica per cluster) |
| **Language** | Go |
| **Connection** | Single bidirectional gRPC stream to Central |
| **Responsibilities** | Kubernetes resource monitoring (informers), event pipeline (listener -> resolver -> output -> detector), deploy-time policy evaluation, runtime detection, enforcement execution, admission control settings distribution, network flow aggregation, process signal processing |
| **Ports** | 8443 (gRPC for Collector + Admission Controller) |

#### Admission Controller

| Aspect | Detail |
|--------|--------|
| **Type** | Stateless service (Deployment, 3 replicas for HA) |
| **Language** | Go |
| **Connection** | gRPC to Sensor for policy sync and alert forwarding |
| **Responsibilities** | Kubernetes ValidatingWebhookConfiguration, admission-time policy evaluation, deployment blocking/allowing, break-glass bypass via annotations |
| **Ports** | 8443 (HTTPS webhook) |

#### Scanner V4

| Aspect | Detail |
|--------|--------|
| **Type** | Stateful service (Deployment, 1+ replicas) |
| **Language** | Go, built on ClairCore |
| **Database** | Separate PostgreSQL 15+ instance |
| **Responsibilities** | Container image indexing (layer download, package extraction, OS detection), vulnerability matching (CVE lookup against 13 ecosystem matchers), CVE enrichment (NVD CVSS, Red Hat CSAF, EPSS), SBOM generation (SPDX 2.3), vulnerability database updates |
| **Ports** | 8443 (gRPC), 9443 (HTTP health/metrics) |

#### Collector

| Aspect | Detail |
|--------|--------|
| **Type** | DaemonSet (one pod per node) |
| **Language** | C++ (userspace), C (eBPF programs) |
| **Connection** | gRPC to Sensor |
| **Responsibilities** | eBPF-based syscall monitoring (execve, connect, accept, close), process signal generation, network connection tracking (ConnTracker), afterglow suppression, delta computation, rate limiting |
| **Repository** | Separate: `github.com/stackrox/collector` |

### External Dependencies

| Dependency | Purpose | Required |
|------------|---------|----------|
| Kubernetes API Server | Resource monitoring, RBAC, admission webhooks | Yes |
| PostgreSQL 15+ | Central and Scanner persistent storage | Yes |
| Container Registries | Image layer download for scanning | Yes (for scanning) |
| `definitions.stackrox.io` | Vulnerability bundle updates | Yes (for up-to-date CVE data) |
| NVD API | CVSS score enrichment | No (bundled in vuln bundles) |
| Red Hat CSAF | Red Hat-specific CVE data | No (bundled in vuln bundles) |

---

## 4. Core Domain Model

### 4.1 Deployment

Represents a Kubernetes workload (Deployment, DaemonSet, StatefulSet, ReplicaSet, Job, CronJob, or bare Pod).

```
Deployment {
  id:                     string (UUID, derived from K8s UID)
  name:                   string
  hash:                   uint64 (change detection hash of spec fields)
  type:                   string ("Deployment" | "DaemonSet" | "StatefulSet" | ...)
  namespace:              string
  namespace_id:           string (UUID)
  cluster_id:             string (UUID)
  cluster_name:           string
  replicas:               int64
  labels:                 map<string, string>
  annotations:            map<string, string>
  pod_labels:             map<string, string>
  created:                Timestamp
  containers:             []Container
  service_account:        string
  service_account_permission_level: PermissionLevel (NONE | DEFAULT | ELEVATED | CLUSTER_ADMIN)
  host_network:           bool
  host_pid:               bool
  host_ipc:               bool
  automount_service_account_token: bool
  tolerations:            []Toleration
  ports:                  []PortConfig (aggregated from containers)
  risk_score:             float32
  priority:               int64
  inactive:               bool (set when deployment deleted from cluster)
  image_pull_secrets:     []string
}

Container {
  id:                     string
  name:                   string
  image:                  ContainerImage
  security_context:       SecurityContext
  resources:              Resources
  volumes:                []Volume
  ports:                  []PortConfig
  config:                 ContainerConfig
  liveness_probe:         Probe
  readiness_probe:        Probe
}

SecurityContext {
  privileged:             bool
  add_capabilities:       []string
  drop_capabilities:      []string
  read_only_root_filesystem: bool
  se_linux:               SELinux
  run_as_user:            int64
  run_as_non_root:        bool
  seccomp_profile:        SeccompProfile
  allow_privilege_escalation: bool
  app_armor_profile:      string
}
```

**Lifecycle:**
1. Sensor Informer detects K8s resource via watch stream
2. Dispatcher converts to `storage.Deployment` proto
3. Resolver enriches with pods, services, RBAC, network policies
4. Detector evaluates deploy-time policies, computes hash
5. Deduper checks hash against last-sent hash; sends to Central if changed
6. Central upserts to PostgreSQL, triggers image enrichment and risk scoring
7. When K8s resource is deleted: `inactive = true`, alerts resolved

**Deduplication:** Four layers of hash-based deduplication:
1. Version tracking (resourceVersion) in Resolver
2. Hash-based skip in Detector
3. Hash-based skip in Deduper (before network transmission)
4. Hash comparison in Central datastore upsert

### 4.2 Image / ImageV2

Represents a container image with metadata and vulnerability scan results.

```
ImageV2 {
  id:                     string (UUID v5 from name + digest)
  name:                   ImageName
  digest:                 string (sha256:...)
  metadata:               ImageMetadata (JSONB)
  scan:                   ImageScan (JSONB)
  signature:              ImageSignature (JSONB)
  signature_verification_data: ImageSignatureVerificationData (JSONB)
  notes:                  []ImageNote (enum array)
  risk_score:             float32
  top_cvss:               float32
  scan_stats:             ScanStats (cached CVE counts)
  created_at:             Timestamp
  last_updated:           Timestamp
}

ImageName {
  registry:               string ("quay.io", "docker.io", ...)
  remote:                 string ("stackrox-io/main")
  tag:                    string ("4.5.0")
  full_name:              string ("quay.io/stackrox-io/main:4.5.0")
}

ImageScan {
  scan_time:              Timestamp
  components:             []EmbeddedImageScanComponent
  operating_system:       string ("rhel:8", "ubuntu:22.04", "alpine:3.18")
  data_source:            DataSource
  notes:                  []ImageScanNote
}

ScanStats {
  cve_count:              int32
  fixable_cve_count:      int32
  critical_cve_count:     int32
  important_cve_count:    int32
  moderate_cve_count:     int32
  low_cve_count:          int32
  component_count:        int32
  top_cvss:               float32
}
```

**Scan States:**
- **Unscanned:** No `scan` field present; image detected but not yet scanned.
- **Scan in progress:** Enricher has initiated scanning; not yet stored.
- **Scanned:** `scan` field populated with components and CVEs.
- **Missing scan data:** `notes` contains `MISSING_SCAN_DATA` (scanner failed).
- **Not operationally scannable:** `notes` contains `NOT_OPERATIONALLY_SCANNABLE`.

**Enrichment Cache Layers:**
1. In-memory metadata cache (TTL: 4 hours)
2. PostgreSQL database check
3. Scanner V4 manifest cache (TTL: 7-30 days, random per manifest)

### 4.3 Alert

Represents a security policy violation.

```
Alert {
  id:                     string (UUID)
  policy:                 Policy (embedded snapshot)
  lifecycle_stage:        LifecycleStage (BUILD | DEPLOY | RUNTIME)
  entity:                 oneof {
                            Deployment deployment
                            ContainerImage image
                            Resource resource
                            Node node
                          }
  violations:             []Violation
  process_violation:      ProcessIndicator (for runtime process alerts)
  enforcement_action:     EnforcementAction
  enforcement_count:      int32
  state:                  ViolationState
  time:                   Timestamp
  first_occurred:         Timestamp
  resolved_at:            Timestamp
  tags:                   []string
  cluster_id:             string
  cluster_name:           string
  namespace:              string
  namespace_id:           string
  snoozed:                bool
  snooze_expiry:          Timestamp
}

Violation {
  message:                string
  type:                   ViolationType
  message_attributes:     MessageAttributes (key-value context)
  key_value_attrs:        KeyValueAttrs
  time:                   Timestamp
}
```

**State Machine:**

```
                    ┌─────────────────┐
  Policy violation  │                 │
  with enforcement──►    ACTIVE       │
                    │                 │
                    └────────┬────────┘
                             │
                    Deployment fixed / deleted /
                    policy disabled / manual resolve
                             │
                    ┌────────▼────────┐
                    │                 │
                    │   RESOLVED      │
                    │                 │
                    └────────┬────────┘
                             │
                    After retention period
                    (configurable, default 30 days)
                             │
                    ┌────────▼────────┐
                    │                 │
                    │   Pruned        │
                    │  (deleted)      │
                    └─────────────────┘


                    ┌─────────────────┐
  Policy violation  │                 │
  inform-only ──────►   ATTEMPTED     │
  (no enforcement)  │                 │
                    └────────┬────────┘
                             │
                    Deployment fixed / deleted
                             │
                    ┌────────▼────────┐
                    │   RESOLVED      │
                    └─────────────────┘
```

**ViolationState enum:**
```
ACTIVE    = 0   // Enforcement was or will be applied
RESOLVED  = 2   // Violation no longer present
ATTEMPTED = 3   // Violation detected, inform-only (no enforcement)
```

Note: Value 1 is reserved (formerly SNOOZED, removed).

**Deduplication Key:** `(policy_id, entity_id, lifecycle_stage)`. When a new violation is detected for an existing key, the violation is appended to the existing alert rather than creating a new alert.

### 4.4 Policy

Defines a security rule with boolean expressions, scoping, and enforcement actions.

```
Policy {
  id:                     string (UUID)
  name:                   string (unique)
  description:            string
  rationale:              string
  remediation:            string
  severity:               Severity (LOW | MEDIUM | HIGH | CRITICAL)
  disabled:               bool
  lifecycle_stages:       []LifecycleStage (BUILD | DEPLOY | RUNTIME)
  event_source:           EventSource (NOT_APPLICABLE | DEPLOYMENT_EVENT | AUDIT_LOG_EVENT)
  policy_sections:        []PolicySection
  enforcement_actions:    []EnforcementAction
  exclusions:             []Exclusion
  scope:                  []Scope
  notifiers:              []string (notifier integration IDs)
  categories:             []string ("DevOps Best Practices", "Security Best Practices", ...)
  mitre_attack_vectors:   []MitreAttackVector
  is_default:             bool
  sort_name:              string
  sort_lifecycle_stage:   string
  policy_version:         string ("1.1")
  criteria_locked:        bool
  mitre_vectors_locked:   bool
}

PolicySection {
  section_name:           string
  policy_groups:          []PolicyGroup
}

PolicyGroup {
  field_name:             string ("Privileged Container", "Image CVE CVSS", ...)
  bool_op:                BooleanOperator (OR | AND)
  negate:                 bool
  values:                 []PolicyValue
}

PolicyValue {
  value:                  string
}

Exclusion {
  name:                   string
  deployment:             Exclusion_Deployment (name regex, scope)
  image:                  Exclusion_Image (name regex)
  expiration:             Timestamp
}

Scope {
  cluster:                string (cluster name or ID)
  namespace:              string (namespace label or name)
  label:                  Scope_Label (key, value)
}
```

**Boolean Expression Structure:**
- A Policy has N PolicySections (OR relationship between sections)
- Each PolicySection has N PolicyGroups (AND relationship within section)
- Each PolicyGroup has N PolicyValues (OR or AND based on `bool_op`)
- A Policy matches if ANY section matches
- A section matches if ALL its groups match

**Lifecycle Stages:**
```
BUILD   = 1   // Evaluated during roxctl image check/scan
DEPLOY  = 0   // Evaluated at deployment creation/update
RUNTIME = 2   // Evaluated against process/network/file/audit events
```

**Enforcement Actions:**
```
UNSET_ENFORCEMENT                    = 0
SCALE_TO_ZERO_ENFORCEMENT            = 2   // Sets replicas to 0
KILL_POD_ENFORCEMENT                  = 4   // Deletes pods
FAIL_BUILD_ENFORCEMENT                = 5   // roxctl exits non-zero
FAIL_KUBE_REQUEST_ENFORCEMENT         = 6   // Admission webhook denies
FAIL_DEPLOYMENT_CREATE_ENFORCEMENT    = 7   // Sensor blocks create
FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT    = 8   // Sensor blocks update
```

### 4.5 Cluster

Represents a registered Kubernetes or OpenShift cluster.

```
Cluster {
  id:                     string (UUID)
  name:                   string (unique)
  type:                   ClusterType (KUBERNETES_CLUSTER | OPENSHIFT4_CLUSTER)
  labels:                 map<string, string>
  main_image:             string (Sensor container image)
  collector_image:        string
  central_api_endpoint:   string
  collection_method:      CollectionMethod (CORE_BPF)
  admission_controller:   bool
  admission_controller_updates: bool
  admission_controller_events: bool
  dynamic_config:         DynamicClusterConfig
  status:                 ClusterStatus
  health_status:          ClusterHealthStatus
  managed_by:             ManagerType (MANUAL | HELM_CHART | KUBERNETES_OPERATOR)
  init_bundle_id:         string
  helm_config:            HelmConfig
}

ClusterHealthStatus {
  id:                     string (FK to Cluster)
  sensor_health_status:   HealthStatusLabel
  collector_health_status: HealthStatusLabel
  overall_health_status:  HealthStatusLabel
  admission_control_health_status: HealthStatusLabel
  scanner_health_status:  HealthStatusLabel
  last_contact:           Timestamp
  health_info_complete:   bool
  collector_health_info:  CollectorHealthInfo
  admission_control_health_info: AdmissionControlHealthInfo
  scanner_health_info:    ScannerHealthInfo
}
```

**HealthStatusLabel:**
```
UNINITIALIZED = 0   // Sensor never connected
UNAVAILABLE   = 1   // Component not deployed (Collector only)
UNHEALTHY     = 2   // Not responding or error state
DEGRADED      = 3   // Partially functional
HEALTHY       = 4   // Fully operational
```

**Health Determination:**
- Sensor health: Based on gRPC connection liveness and heartbeat messages
- Collector health: Reported by Sensor from Collector gRPC connection
- Admission Controller health: Reported by Sensor from AC gRPC connection
- Scanner health: Reported by Sensor from local Scanner health checks
- Overall health: `min(sensor, collector, admission_control, scanner)` using ordering above

### 4.6 NetworkFlow

Represents an observed network connection between two entities.

```
NetworkFlow {
  props:                  NetworkFlowProperties
  last_seen_timestamp:    Timestamp (null if still active)
  cluster_id:             string
}

NetworkFlowProperties {
  src_entity:             NetworkEntityInfo
  dst_entity:             NetworkEntityInfo
  dst_port:               uint32
  l4protocol:             L4Protocol (L4_PROTOCOL_TCP | L4_PROTOCOL_UDP | ...)
}

NetworkEntityInfo {
  type:                   Type (DEPLOYMENT | EXTERNAL_SOURCE | INTERNET | INTERNAL_ENTITIES)
  id:                     string
  // If DEPLOYMENT: id = deployment UUID
  // If EXTERNAL_SOURCE: id = cluster-scoped ID "<clusterID>:<base64(CIDR)>"
  // If INTERNET: sentinel value
}
```

**Storage:** Partitioned PostgreSQL table per cluster. Primary key is `(cluster_id, src_entity_type, src_entity_id, dst_entity_type, dst_entity_id, dst_port, protocol)`.

**Semantics:**
- `last_seen_timestamp = null` means the flow is currently active
- `last_seen_timestamp != null` means the flow was last seen at that time and is now closed
- Flows are upserted: same connection tuple updates the timestamp
- Terminated flows are retained for configurable period (default 7 days) then pruned

### 4.7 ProcessIndicator

Represents a process execution detected by Collector.

```
ProcessIndicator {
  id:                     string (UUID v5 deterministic from components)
  deployment_id:          string (UUID)
  container_name:         string
  pod_id:                 string
  pod_uid:                string
  namespace:              string
  image_id:               string
  cluster_id:             string (UUID)
  signal:                 ProcessSignal
  container_start_time:   Timestamp
}

ProcessSignal {
  id:                     string
  container_id:           string
  time:                   Timestamp
  name:                   string (process name, e.g. "nginx")
  args:                   string (command line arguments)
  exec_file_path:         string (e.g. "/usr/sbin/nginx")
  pid:                    uint32
  uid:                    uint32
  gid:                    uint32
  lineage_info:           []LineageInfo (parent processes)
  scraped:                bool
}
```

**Stable ID Generation:**
```
id = UUID_v5(
  namespace = processIDNamespace,
  components = [pod_id, container_name, exec_file_path, name, args]
)
```

This ensures idempotent upserts and deduplication across restarts.

**Data Retention:** Default 7 days, configurable via `ROX_PROCESS_PRUNING_RETENTION`.

### 4.8 Role / PermissionSet / SimpleAccessScope

StackRox's internal RBAC model (not to be confused with Kubernetes RBAC).

```
Role {
  name:                   string (unique, primary key)
  description:            string
  permission_set_id:      string (FK to PermissionSet)
  access_scope_id:        string (FK to SimpleAccessScope)
  traits:                 Traits (ALLOW_MUTATE | IMPERATIVE for system roles)
}

PermissionSet {
  id:                     string (UUID)
  name:                   string (unique)
  description:            string
  resource_to_access:     map<string, Access>
  // Key: resource name (e.g. "Deployment", "Alert", "Image")
  // Value: READ_ACCESS | READ_WRITE_ACCESS
  traits:                 Traits
}

SimpleAccessScope {
  id:                     string (UUID)
  name:                   string (unique)
  description:            string
  rules:                  Rules
  traits:                 Traits
}

SimpleAccessScope.Rules {
  included_clusters:      []string (cluster names)
  included_namespaces:    []Namespace (cluster + namespace pairs)
  cluster_label_selectors: []SetBasedLabelSelector
  namespace_label_selectors: []SetBasedLabelSelector
}

Access enum {
  NO_ACCESS         = 0
  READ_ACCESS       = 1
  READ_WRITE_ACCESS = 2
}
```

**Composition:** A Role combines a PermissionSet (what operations on which resources) with a SimpleAccessScope (which clusters and namespaces). A user may have multiple roles; effective access is the union (OR) of all role grants.

### 4.9 CVE / ImageCVEV2

```
ImageCVEV2 {
  id:                     string (composite: "imageID:cveID")
  cve_base_info:          CVEInfo
  cvss:                   float32 (ACS-preferred CVSS score)
  severity:               VulnerabilitySeverity
  is_fixable:             bool
  fixed_by:               string (version that fixes the CVE)
  component_id:           string (FK to ImageComponentV2)
  image_id_v2:            string (FK to ImageV2)
  state:                  VulnerabilityState (OBSERVED | DEFERRED | FALSE_POSITIVE)
}

CVEInfo {
  cve:                    string ("CVE-2023-1234")
  summary:                string
  link:                   string
  published_on:           Timestamp
  created_at:             Timestamp
  last_modified:          Timestamp
  cvss_metrics:           []CVSSScore
  epss:                   EPSS (score float32, percentile float32)
}

VulnerabilitySeverity enum {
  UNKNOWN_VULNERABILITY_SEVERITY = 0
  LOW_VULNERABILITY_SEVERITY     = 1
  MODERATE_VULNERABILITY_SEVERITY = 2
  IMPORTANT_VULNERABILITY_SEVERITY = 3
  CRITICAL_VULNERABILITY_SEVERITY = 4
}
```

### 4.10 ComplianceOperatorProfileV2 / ComplianceOperatorCheckResultV2

```
ComplianceOperatorProfileV2 {
  id:                     string
  name:                   string
  profile_version:        string
  product_type:           string ("ocp4", "rhcos4")
  standard:               string ("CIS", "PCI-DSS", "NIST-800-53")
  description:            string
  rules:                  []Rule
  title:                  string
  values:                 []string
  annotations:            map<string, string>
}

ComplianceOperatorCheckResultV2 {
  id:                     string
  check_id:               string
  check_name:             string
  cluster_id:             string
  status:                 CheckStatus (PASS | FAIL | ERROR | INFO | MANUAL |
                                       NOT_APPLICABLE | INCONSISTENT)
  severity:               RuleSeverity
  description:            string
  instructions:           string
  labels:                 map<string, string>
  annotations:            map<string, string>
  created_time:           Timestamp
  scan_name:              string
  scan_config_name:       string
  rationale:              string
  valuesused:             []string
  warnings:               []string
}
```

### 4.11 NetworkBaseline / ProcessBaseline

```
NetworkBaseline {
  deployment_id:          string (primary key)
  cluster_id:             string
  namespace:              string
  peers:                  []NetworkBaselinePeer
  forbidden_peers:        []NetworkBaselinePeer
  observation_period_end: Timestamp
  locked:                 bool
  deployment_name:        string
}

NetworkBaselinePeer {
  entity:                 NetworkEntity
  properties:             []NetworkBaselineConnectionProperties
}

NetworkBaselineConnectionProperties {
  incoming:               bool
  port:                   uint32
  protocol:               L4Protocol
}

ProcessBaseline {
  id:                     string
  key:                    ProcessBaselineKey (deployment_id, container_name, cluster_id, namespace)
  elements:               []BaselineElement (process name set)
  element_graveyard:      []BaselineElement (explicitly removed)
  created:                Timestamp
  last_update:            Timestamp
  stack_rox_locked_timestamp: Timestamp
  user_locked_timestamp:  Timestamp
}
```

**Baseline States:**
- **Learning (Unlocked):** Processes/connections observed are automatically added. No violations generated.
- **User-Locked:** User explicitly locked the baseline. Violations generated for unrecognized processes/connections.
- **Auto-Locked:** System automatically locked after observation period (default 24 hours for network, configurable for process).

---

## 5. Communication Protocol

### 5.1 Central <-> Sensor gRPC Stream

**Service Definition:**
```protobuf
service SensorService {
  rpc Communicate(stream MsgFromSensor) returns (stream MsgToSensor);
}
```

This is a single bidirectional gRPC stream per cluster. All communication between Central and Sensor flows through this stream.

### 5.2 Handshake Sequence

```
Sensor                                Central
  │                                      │
  │──── SensorHello ─────────────────────►│
  │     {                                 │
  │       sensor_version: "4.5.0"         │
  │       capabilities: [...]             │
  │       sensor_state: STARTUP|RECONNECT │
  │       policy_version: "1.1"           │
  │       deployment_identification: {...} │
  │       request_deduper_state: true     │
  │     }                                 │
  │                                      │
  │◄──── CentralHello ──────────────────│
  │     {                                 │
  │       cluster_id: "uuid"              │
  │       cert_bundle: {ca: "...", ...}   │
  │       capabilities: [...]             │
  │       send_deduper_state: true|false  │
  │       central_id: "uuid"             │
  │       allowed_proxy_paths: [...]      │
  │     }                                 │
  │                                      │
  │◄──── PolicySync ─────────────────────│
  │◄──── BaselineSync ───────────────────│
  │◄──── NetworkBaselineSync ────────────│
  │◄──── ClusterConfig ──────────────────│
  │                                      │
  │──── SensorEvent(SYNC) ──────────────►│  (initial resource sync)
  │──── SensorEvent(SYNC) ──────────────►│
  │     ...                               │
  │──── SyncCompleted ──────────────────►│
  │                                      │
```

### 5.3 Message Types: Sensor -> Central

| Message | Content | Trigger |
|---------|---------|---------|
| `SensorHello` | Version, capabilities, state | Connection establishment |
| `SensorEvent` | K8s resource change (Deployment, Pod, Node, NetworkPolicy, RBAC, etc.) | Informer watch event |
| `NetworkFlowUpdate` | Batch of network flow deltas | Periodic (30s interval) |
| `ClusterStatusUpdate` | Cluster orchestrator metadata | Status change |
| `ClusterHealthInfo` | Component health reports | Periodic heartbeat |
| `ProcessListeningOnPortsUpdate` | Processes bound to ports | Process detection |
| `ComplianceOperatorInfo` | Compliance scan results | Scan completion |
| `AuditLogStatusInfo` | Audit log collection status | Status change |
| `NodeInventory` | Node OS/package inventory | Node scan |

### 5.4 Message Types: Central -> Sensor

| Message | Content | Trigger |
|---------|---------|---------|
| `CentralHello` | Cluster ID, cert bundle, capabilities | Response to SensorHello |
| `PolicySync` | Full set of compiled policies | Policy create/update/delete |
| `BaselineSync` | Process baselines for deployments | Baseline change |
| `NetworkBaselineSync` | Network baselines for deployments | Baseline change |
| `ClusterConfig` | Dynamic cluster configuration | Config change |
| `ReprocessDeployment` | Request to re-evaluate specific deployment | Policy change, image rescan |
| `InvalidateImageCache` | Clear cached image data | New scan results |
| `UpdatedImage` | New image scan data | Scan completion |
| `DelegatedRegistryConfig` | Registry scanning delegation config | Config change |
| `SensorUpgradeTrigger` | Initiate Sensor upgrade | Operator/admin action |

### 5.5 Capability Negotiation

Both SensorHello and CentralHello include a `capabilities` field (repeated string). Capabilities are additive feature flags that allow version-skewed components to negotiate supported features. If Sensor advertises a capability that Central does not recognize, Central ignores it. If Central advertises a capability Sensor does not recognize, Sensor ignores it.

### 5.6 Reconnection Semantics

**Sensor behavior on disconnection:**
1. Enter offline mode (continue K8s monitoring, cache policies, buffer events)
2. Retry connection with exponential backoff:
   - Initial interval: 5 seconds (env: `ROX_CONNECTION_RETRY_INITIAL_INTERVAL`)
   - Maximum interval: 5 minutes (env: `ROX_CONNECTION_RETRY_MAX_INTERVAL`)
   - Infinite retries (never gives up)
3. On reconnection:
   - Send SensorHello with `sensor_state = RECONNECT`
   - If `request_deduper_state = true`, Central sends its known resource hashes
   - Sensor compares local hashes, sends only changed resources
   - Flush buffered alerts and events (within expiration time)
   - Resume runtime event processing (unpause queues)

**Bounded Queues in Offline Mode:**
- Process indicator queue: 10,000 capacity (pausable)
- Network flow queue: 10,000 capacity (pausable)
- File access queue: 1,000 capacity (pausable)
- When queues are full, oldest events are dropped

### 5.7 mTLS Certificate Model

All inter-component communication uses mutual TLS. See Section 16 for certificate hierarchy details.

---

## 6. Data Pipelines

### 6.1 Image Scanning Pipeline

```
TRIGGER
  │
  ├── Deployment detected (Sensor sends new image reference)
  ├── Manual scan (roxctl image scan / UI scan)
  ├── Watched image (periodic rescan)
  │
  ▼
IMAGE NAME PARSING
  │  Input:  "quay.io/stackrox-io/main:4.5.0" or "nginx@sha256:abc..."
  │  Output: ImageName { registry, remote, tag, digest }
  │
  ▼
ENRICHMENT CACHE CHECK (multi-layer)
  │
  │  Layer 1: In-memory metadata cache
  │    Key:   image ID (sha256) or full name
  │    TTL:   4 hours
  │    Hit:   skip metadata fetch, proceed to scan check
  │
  │  Layer 2: PostgreSQL database check
  │    Query: SELECT * FROM images_v2 WHERE id = $1
  │    Hit:   if metadata valid AND scan present AND FetchOpt allows cache → return
  │
  │  Miss: continue to scanner delegation
  │
  ▼
SCANNER DELEGATION DECISION
  │
  │  if delegable AND cluster has delegated_registry_config:
  │    delegate scan to Sensor's local Scanner V4
  │  else:
  │    scan at Central's Scanner V4
  │
  ▼
SCANNER V4 INDEX PHASE
  │  Input:  CreateIndexReportRequest { hash_id, image_name, credentials }
  │
  │  1. Authenticate to container registry
  │  2. Fetch OCI/Docker manifest
  │  3. For each layer:
  │     a. Download (Range: bytes=0-0 optimization if supported)
  │     b. Extract tar contents
  │     c. Run ecosystem scanners:
  │        - Alpine:  /lib/apk/db/installed
  │        - Debian:  /var/lib/dpkg/status
  │        - RPM:     /var/lib/rpm/Packages
  │        - Go:      binary version info
  │        - Java:    .jar/.war files
  │        - Python:  site-packages/
  │        - Node:    node_modules/
  │        - Ruby:    gems/
  │  4. Detect OS from /etc/os-release, /etc/redhat-release, etc.
  │  5. Store IndexReport in Scanner PostgreSQL
  │     - manifest table (hash -> claircore.IndexReport)
  │     - manifest_metadata table (hash -> expiry 7-30 days random)
  │
  │  Output: IndexReport { hash, packages[], distributions[], environments[] }
  │
  ▼
SCANNER V4 MATCH PHASE
  │  Input:  GetVulnerabilitiesRequest { hash_id }
  │
  │  1. Retrieve IndexReport from PostgreSQL
  │  2. For each package:
  │     a. Select matcher by ecosystem (Alpine, Debian, RHEL, Ubuntu, SUSE, ...)
  │     b. Query vulnerability DB:
  │        WHERE package_name = pkg.name
  │          AND distribution matches
  │          AND version in affected range
  │     c. Collect matching CVEs
  │  3. For each matched CVE, run enrichers:
  │     a. NVD enricher → CVSS v2/v3.0/v3.1 scores
  │     b. CSAF enricher → Red Hat CVSS, fixed-in versions
  │     c. EPSS enricher → exploit prediction score
  │     d. FixedBy enricher → fixed version determination
  │
  │  Output: VulnerabilityReport { contents, vulnerabilities, enrichments }
  │
  ▼
CONVERSION TO STACKROX FORMAT
  │  Input:  VulnerabilityReport
  │
  │  1. Create ImageComponentV2 records from packages
  │  2. Create ImageCVEV2 records from vulnerabilities
  │  3. Compute ScanStats (CVE counts by severity, top CVSS)
  │  4. Build ImageV2 proto
  │
  ▼
STORAGE
  │  Transaction:
  │    1. UPSERT images_v2
  │    2. DELETE + BATCH INSERT image_cves_v2
  │    3. UPSERT image_components_v2
  │    4. UPSERT image_cve_info (lookup table)
  │
  ▼
RISK CALCULATION
  │  See Section 13
  │
  ▼
POLICY EVALUATION
  │  Evaluate all BUILD/DEPLOY lifecycle policies against enriched image
  │  See Section 8
```

**Performance Characteristics:**
- New image (no cache): 30s - 5min (depends on size, layers)
- Cached index, new vulns: 5-15s (matcher only)
- Fully cached: <100ms (database lookup)
- Max concurrent scans: controlled by semaphore (default: 10)

### 6.2 Runtime Detection Pipeline

```
KERNEL (eBPF)
  │
  │  CO-RE kprobe on sys_execve:
  │    - timestamp (ns), pid, tid, ppid, uid, gid
  │    - comm (16 chars), filename (256 chars), args (512 chars)
  │    - container_id (from cgroup)
  │  Overhead: <10us per execve
  │  Delivery: BPF ring buffer to userspace
  │
  ▼
COLLECTOR (C++ userspace)
  │
  │  ProcessSignalHandler:
  │    1. Read from BPF ring buffer
  │    2. Enrich with container metadata
  │    3. Build process tree (lineage)
  │    4. Rate limit: 100 signals/sec per container
  │       FILTER: Excess signals dropped
  │    5. Batch: up to 100 signals per gRPC message, flush every 5s
  │
  │  gRPC stream to Sensor (SignalService/PushSignals)
  │
  ▼
SENSOR (Go)
  │
  │  Process Signal Pipeline:
  │    1. Receive ProcessSignal batch
  │    2. Create ProcessIndicator wrapper
  │    3. Enrich with deployment context:
  │       - container_id → pod_id → deployment_id
  │       - Cache lookup in clusterEntities store
  │       - If cache miss: async enrichment queue
  │    4. Generate stable ID (UUID v5)
  │    5. Normalize UTF-8
  │
  ▼
  │  Process Baseline Comparison:
  │    - Check if exec_file_path in locked baseline
  │    - Set outside_baseline flag (does not drop)
  │
  ▼
  │  Similarity-Based Filter:
  │    - Tree structure per deployment+container
  │    - Levels correspond to argument depth
  │    - Fan-out limits per level: [8, 6, 4, 2] (default)
  │    - Max exact path matches: 10 (default)
  │    - FILTER: ~40-50% of processes dropped (default level)
  │
  │    Filter levels:
  │      Aggressive: MaxExact=5,  FanOut=[4,3,2,1]  → ~60-70% dropped
  │      Default:    MaxExact=10, FanOut=[8,6,4,2]   → ~40-50% dropped
  │      Minimal:    MaxExact=50, FanOut=[16,12,8,4] → ~20-30% dropped
  │
  ▼
  │  Runtime Policy Evaluation:
  │    - For each RUNTIME policy with matching scope:
  │      evaluate against ProcessIndicator
  │    - Generate alerts for violations
  │    - Send alerts to Central
  │
  ▼
CENTRAL (Go)
  │
  │  Lifecycle Manager:
  │    1. Queue indicators (in-memory)
  │    2. Rate-limited flush (5 flushes per 10 seconds)
  │    3. Batch upsert to process_indicators table
  │
  │  Risk Recalculation:
  │    - Process baseline violations multiplier
  │    - Update deployment risk score
  │
  │  Data Retention:
  │    - Daily pruning job
  │    - Delete indicators older than retention period (default 7 days)
```

### 6.3 Network Flow Pipeline

```
KERNEL (eBPF)
  │
  │  Hooks: sys_connect, sys_accept4, sys_close
  │  Captures: src/dst IP:port, direction, protocol, container_id, process info
  │
  ▼
COLLECTOR: ConnTracker
  │
  │  1. Connection state tracking: INITIATED → ESTABLISHED → CLOSED
  │  2. Normalization: raw syscall data → NetworkConnection
  │  3. Filter: loopback connections (127.0.0.0/8, ::1) → DROPPED
  │  4. Afterglow suppression:
  │     - Connection closes → move to afterglow set (30s default)
  │     - If reopens during afterglow → restore to active
  │     - If afterglow expires → report as CLOSED
  │     - FILTER: Transient connections never sent to Sensor
  │  5. Delta computation:
  │     - Compare current snapshot vs previous
  │     - Send only new/closed connections
  │     - FILTER: Unchanged connections not sent
  │  6. Rate limiting: max 1000 connections/container/interval
  │     - FILTER: Excess dropped
  │
  │  Batched every 30s, gRPC to Sensor
  │
  ▼
SENSOR: NetworkFlowManager
  │
  │  1. Container → Deployment resolution
  │     - container_id → pod_id → deployment_id
  │     - FILTER: Orphaned containers (deleted pods) dropped
  │  2. External IP classification:
  │     - Private IPs → DEPLOYMENT entity
  │     - Public IPs → EXTERNAL_SOURCE (discovered) or INTERNET
  │     - FILTER: Loopback dropped (double-check)
  │  3. Network flow update computation:
  │     - Transition-based: new connections, closed connections
  │     - FILTER: Invalid endpoints (nil src/dst) dropped
  │  4. Buffered queue to Central (default 10,000)
  │     - FILTER: Queue overflow → dropped
  │
  │  MsgFromSensor { NetworkFlowUpdate { flows[], time, sequence_id } }
  │
  ▼
CENTRAL: Flow Pipeline
  │
  │  1. Entity ID fixup for discovered externals:
  │     - Assign cluster-scoped ID: "<clusterID>:<base64(CIDR)>"
  │  2. Feature flag: ExternalIPs
  │     - Disabled: all discovered externals → INTERNET sentinel
  │  3. Upsert flows to PostgreSQL (partitioned per cluster)
  │  4. Mark stale flows as terminated (first update after Sensor restart)
  │
  ▼
CENTRAL: Aggregation Pipeline (for graph queries)
  │
  │  1. Subnet-to-supernet: discovered IPs → containing custom external source
  │  2. Default-to-custom: default external sources (AWS, GCP) → custom supernets
  │  3. Duplicate name: multiple external sources with same name → single node
  │  4. Latest timestamp: deduplicate identical flows, keep most recent
  │
  ▼
CENTRAL: Network Baseline
  │
  │  1. For each flow, check if peer is in deployment's baseline
  │  2. If baseline locked and peer not in baseline → generate alert
  │  3. If baseline learning → add peer to baseline
  │
  ▼
CENTRAL: Network Policy Evaluation
  │
  │  1. Build network graph from deployments + K8s NetworkPolicies
  │  2. Annotate flows with policy status (ALLOWED | DENIED)
  │  3. Serve via NetworkGraphService
```

### 6.4 Policy Evaluation Pipeline

```
POLICY DEFINITION (UI/API/roxctl)
  │
  │  1. Validate policy structure
  │  2. Assign UUID if new
  │  3. Store in PostgreSQL
  │  4. Notify detection engine
  │
  ▼
POLICY COMPILATION (pkg/booleanpolicy)
  │
  │  1. For each PolicySection:
  │     a. Parse field_name → field path through object model
  │     b. Compile matcher based on field type:
  │        - Bool matcher: exact equality
  │        - String matcher: regex compilation
  │        - Numeric matcher: comparison operators (>=, <=, ==, !=, >, <)
  │        - Enum matcher: set membership
  │        - Map matcher: key/value matching
  │        - Timestamp matcher: age/duration comparison
  │     c. Build linked groups (ensure matches from same sub-object)
  │  2. Construct boolean expression tree
  │  3. Cache compiled policy in PolicySet (in-memory)
  │  4. Distribute to Sensor via PolicySync message
  │
  ▼
EVALUATION (context-dependent)
  │
  ├── BUILD TIME (Central):
  │   │  Trigger: roxctl image check/scan
  │   │  Context: Image only (no deployment)
  │   │  Available fields: image name, CVEs, Dockerfile, image config
  │   │  Enforcement: FAIL_BUILD_ENFORCEMENT → roxctl exits non-zero
  │   │
  │
  ├── DEPLOY TIME (Sensor Detector + Admission Controller):
  │   │  Trigger: Deployment create/update from K8s informer or admission webhook
  │   │  Context: Full deployment + enriched image scan data
  │   │  Available fields: all image fields + container config + security context
  │   │                     + RBAC + network policies + labels + annotations
  │   │  Scoping filter:
  │   │    1. Check cluster scope
  │   │    2. Check namespace scope
  │   │    3. Check label selectors
  │   │    4. Check exclusions (deployment name, image name, expiration)
  │   │  Enforcement: FAIL_KUBE_REQUEST, SCALE_TO_ZERO, KILL_POD
  │   │
  │   │  Admission Controller specifics:
  │   │    - Separate pod from Sensor
  │   │    - Cached policies (survives Sensor restart)
  │   │    - Fail-open on error (allow deployment)
  │   │    - Break-glass bypass annotation:
  │   │      admission.stackrox.io/break-glass: "ticket-1234"
  │   │
  │
  └── RUNTIME (Sensor Detector + Central Lifecycle Manager):
      │  Trigger: Process execution, network connection, file access, audit log
      │  Context: Deployment + runtime event
      │  Available fields: process name/path/args/uid, network dest,
      │                     file path, K8s API operation
      │  Baseline integration:
      │    - Process baseline: IsOutsideLockedBaseline(execPath)
      │    - Network baseline: peer not in locked baseline
      │  Enforcement: KILL_POD
      │
  ▼
ALERT GENERATION (AlertManager)
  │
  │  1. Deduplication by (policy_id, entity_id, lifecycle_stage)
  │     - Existing alert: append violations, update timestamp
  │     - New alert: create with ViolationCount=1
  │  2. State determination:
  │     - Enforcement action present → ACTIVE
  │     - Inform only → ATTEMPTED
  │  3. Store in PostgreSQL alerts table
  │  4. Route to notifiers
  │
  ▼
NOTIFICATION (NotifierProcessor)
  │
  │  1. For each alert, get policy's notifier IDs
  │  2. For each notifier:
  │     a. Check label-based filter (severity, namespace)
  │     b. Format alert message (template-based)
  │     c. Send notification (async, non-blocking)
  │     d. Retry on failure (3 attempts, 10s timeout)
  │
  ▼
ENFORCEMENT (Sensor Enforcer)
  │
  │  Executed in Sensor (has K8s API access):
  │  - SCALE_TO_ZERO: Set deployment replicas to 0
  │    (original replica count saved in annotation stackrox.io/original-replicas)
  │  - KILL_POD: Delete all pods matching deployment's pod labels
  │  - FAIL_KUBE_REQUEST: Admission webhook returns Allowed: false
  │  - FAIL_DEPLOYMENT_CREATE/UPDATE: Sensor blocks via K8s API
```

### 6.5 Deployment Discovery Pipeline

```
K8s API Server
  │
  │  Watch stream (client-go SharedInformer)
  │
  ▼
LISTENER (sensor/kubernetes/listener/)
  │
  │  Resource types monitored:
  │    Workloads: Deployment, DaemonSet, StatefulSet, ReplicaSet, Pod, Job, CronJob
  │    Networking: Service, NetworkPolicy, Ingress, Route (OpenShift)
  │    RBAC: Role, RoleBinding, ClusterRole, ClusterRoleBinding, ServiceAccount
  │    Config: Secret, ConfigMap, Namespace
  │    Nodes, ComplianceOperator CRs, VirtualMachine (KubeVirt)
  │
  │  Per resource type:
  │    1. Dispatcher converts K8s object → storage proto
  │    2. Updates local in-memory store
  │    3. Generates ResourceEvent with deployment references
  │
  │  Action mapping:
  │    Initial sync → SYNC_RESOURCE
  │    New resource → CREATE_RESOURCE
  │    Updated resource → UPDATE_RESOURCE
  │    Deleted resource → REMOVE_RESOURCE
  │
  │  FILTER: Large annotations truncated (e.g., last-applied-configuration)
  │
  ▼
RESOLVER (sensor/kubernetes/eventpipeline/resolver/)
  │
  │  For each deployment reference:
  │    1. Fetch deployment from local store
  │    2. Resolve parent (ReplicaSet → owning Deployment)
  │    3. Gather pods from pod store
  │    4. Resolve services (label selector matching)
  │    5. Find applied network policies
  │    6. Resolve RBAC:
  │       - ServiceAccount → RoleBindings → Roles → PolicyRules
  │       - Compute PermissionLevel (NONE | DEFAULT | ELEVATED | CLUSTER_ADMIN)
  │
  │  FILTER: ResourceVersion deduplication (skip if unchanged)
  │
  ▼
OUTPUT QUEUE → DETECTOR (sensor/common/detector/)
  │
  │  1. Hash-based deduplication:
  │     - Compute hash of deployment spec (excluding volatile fields)
  │     - Compare against last-processed hash
  │     - FILTER: Skip if hash unchanged
  │
  │  2. Deploy-time policy evaluation:
  │     - Enrich with image scan data from Central
  │     - Run all DEPLOY lifecycle policies
  │     - Generate alerts for violations
  │     - Invoke enforcer if enforcement configured
  │
  ▼
DEDUPER (sensor/common/deduper/)
  │
  │  Final hash-based check before transmission:
  │    - SensorHash field set on each event
  │    - FILTER: Skip if hash matches last-sent value
  │
  │  MsgFromSensor { SensorEvent { id, action, Deployment, DeploymentAlerts } }
  │
  ▼
CENTRAL
  │
  │  1. Deployment upsert:
  │     - Keyed mutex on deployment ID
  │     - Hash comparison (skip if unchanged)
  │     - Store in PostgreSQL deployments table
  │
  │  2. Image enrichment trigger:
  │     - For each container image not yet scanned → enqueue scan
  │
  │  3. Risk calculation:
  │     - See Section 13
  │
  │  4. Alert processing:
  │     - Store alerts from Sensor
  │     - Route to notifiers
```

**Total deduplication layers: 4** (Resolver version tracking, Detector hash, Deduper hash, Central hash). In steady state, approximately 80% of deployment updates are suppressed before reaching Central.

---

## 7. Authorization Model (SAC)

### 7.1 Three-State Logic

Every access check returns one of three states:

| State | Meaning |
|-------|---------|
| **Excluded** | No access to this scope or any descendant. The request is denied. |
| **Partial** | Access to some but not all children. Query filters must be applied. |
| **Included** | Full access to this scope and all descendants. No filtering needed. |

### 7.2 Scope Hierarchy

```
GlobalScope
  └── AccessModeScope (READ_ACCESS | READ_WRITE_ACCESS)
      └── ResourceScope (Alert, Deployment, Image, Cluster, Node, ...)
          └── ClusterScope (cluster ID)
              └── NamespaceScope (namespace name)
```

Access checks traverse the hierarchy top-down. At each level, the checker returns Excluded, Partial, or Included:

```
function check(scope_path):
  state = root
  for key in scope_path:
    state = state.SubScopeChecker(key)
    if state.Allowed():
      return Included  // all descendants included
  return state  // Partial or Excluded
```

### 7.3 ScopeChecker Interface

```go
type ScopeChecker interface {
  // Navigate to sub-scope
  AccessMode(am storage.Access) ScopeChecker
  Resource(resource permissions.ResourceHandle) ScopeChecker
  ClusterID(clusterID string) ScopeChecker
  Namespace(namespace string) ScopeChecker

  // Check access
  IsAllowed(subScopeKeys ...ScopeKey) bool

  // Compute effective access scope (for query filtering)
  EffectiveAccessScope(resource permissions.ResourceWithAccess) (*ScopeTree, error)
}
```

### 7.4 Effective Access Scope Tree

The ScopeTree represents the computed intersection of user permissions with actual cluster/namespace topology:

```
ScopeTree {
  State:    Excluded | Partial | Included
  Clusters: map[clusterName] -> {
    State:      Excluded | Partial | Included
    Namespaces: map[namespaceName] -> {
      State: Excluded | Included
    }
  }
}
```

**Computation:**
1. Start with user's SimpleAccessScope rules
2. Expand label selectors against actual cluster/namespace labels
3. Intersect included clusters/namespaces with role's PermissionSet for the requested resource
4. Result is a tree showing which clusters/namespaces the user can access

### 7.5 Query Filter Injection

SAC filters are injected into database queries automatically:

**Cluster-scoped resources** (Node, Cluster):
```sql
WHERE cluster_id IN ('cluster-1', 'cluster-2')
```

**Namespace-scoped resources** (Deployment, Alert, Image):
```sql
WHERE (cluster_id = 'cluster-1' AND namespace IN ('ns-a', 'ns-b'))
   OR (cluster_id = 'cluster-2')  -- full access to cluster-2
```

If the ScopeTree state is Included (full access), no filter is injected. If Excluded, the query returns empty results without hitting the database.

### 7.6 Resource Scoping Levels

| Level | Resources |
|-------|-----------|
| **Global** | Access, Administration, Integration, InstallationInfo |
| **Cluster** | Cluster, Node, Compliance |
| **Namespace** | Deployment, Alert, Image, NetworkPolicy, Secret, ServiceAccount, K8sRole, K8sRoleBinding, NetworkGraph, ProcessWhitelist |

### 7.7 Context Propagation

SAC scope checkers are propagated via `context.Context`:

1. gRPC/HTTP interceptor extracts user identity from auth token or mTLS certificate
2. Interceptor resolves user's roles → PermissionSets + AccessScopes
3. Creates ScopeChecker from union of all role grants
4. Attaches ScopeChecker to context: `sac.WithGlobalAccessScopeChecker(ctx, checker)`
5. Service layer retrieves: `sac.GlobalAccessScopeChecker(ctx)`
6. Datastore layer uses EffectiveAccessScope to build query filters

---

## 8. Policy Engine Specification

### 8.1 Boolean Expression Structure

```
Policy
  └── PolicySection[] (OR - any section match = policy match)
      └── PolicyGroup[] (AND - all groups must match within section)
          ├── field_name: string
          ├── bool_op: OR | AND (applied to values within group)
          ├── negate: bool (invert match result)
          └── PolicyValue[] (combined per bool_op)
              └── value: string
```

**Pseudocode:**
```
function matches(policy, object):
  for section in policy.policy_sections:
    if all_groups_match(section.policy_groups, object):
      return true
  return false

function all_groups_match(groups, object):
  for group in groups:
    result = evaluate_group(group, object)
    if group.negate:
      result = !result
    if !result:
      return false
  return true

function evaluate_group(group, object):
  field_values = extract_field(object, group.field_name)
  for obj_value in field_values:  // existential quantification
    matched = match_values(group.values, obj_value, group.bool_op)
    if matched:
      return true
  return false

function match_values(policy_values, obj_value, bool_op):
  if bool_op == OR:
    return any(v.matches(obj_value) for v in policy_values)
  else:  // AND
    return all(v.matches(obj_value) for v in policy_values)
```

### 8.2 Field Metadata Registry

Each policy field name maps to an object path, type, and matcher:

| Field Name | Object Path | Type | Matcher |
|-----------|-------------|------|---------|
| `Privileged Container` | `deployment.containers[*].securityContext.privileged` | bool | equality |
| `Image CVE CVSS` | `image.scan.components[*].vulns[*].cvss` | numeric | comparison (>=, <=) |
| `Process Name` | `processIndicator.signal.name` | string | regex |
| `Process Baseline` | computed | enum | NOT_IN_BASELINE |
| `Image Component` | `image.scan.components[*].name` | string | regex |
| `Fixable CVE` | `image.scan.components[*].vulns[*].fixedBy` | bool | non-empty check |
| `Image Registry` | `image.name.registry` | string | regex |
| `Image Tag` | `image.name.tag` | string | regex |
| `Environment Variable` | `deployment.containers[*].config.env[*]` | map | key=value |
| `Add Capabilities` | `deployment.containers[*].securityContext.addCapabilities[*]` | string | regex |
| `Drop Capabilities` | `deployment.containers[*].securityContext.dropCapabilities[*]` | string | regex |
| `Read-Only Root Filesystem` | `deployment.containers[*].securityContext.readOnlyRootFilesystem` | bool | equality |
| `Mount Propagation` | `deployment.containers[*].volumes[*].mountPropagation` | enum | equality |
| `Volume Name` | `deployment.containers[*].volumes[*].name` | string | regex |
| `Volume Type` | `deployment.containers[*].volumes[*].type` | string | regex |
| `Port` | `deployment.ports[*].containerPort` | numeric | comparison |
| `Protocol` | `deployment.ports[*].protocol` | string | equality |
| `Exposed Port` | `deployment.ports[*].exposure` | enum | UNSET/INTERNAL/NODE/EXTERNAL/ROUTE |
| `Namespace` | `deployment.namespace` | string | regex |
| `Cluster` | `deployment.clusterName` | string | regex |
| `Label` | `deployment.labels` | map | key=value |
| `Annotation` | `deployment.annotations` | map | key=value |
| `Service Account` | `deployment.serviceAccount` | string | regex |
| `Replicas` | `deployment.replicas` | numeric | comparison |
| `Minimum RBAC Permissions` | `deployment.serviceAccountPermissionLevel` | enum | comparison |
| `Image Age` | computed from `image.metadata.created` | duration | comparison |
| `Unscanned Image` | computed from `image.scan` | bool | presence check |
| `Image Signature Verified By` | `image.signatureVerificationData` | string | equality |
| `Network Baseline` | computed | enum | NOT_IN_BASELINE |
| `Writable Mounted Volume` | computed | bool | volume write check |
| `Automount Service Account Token` | `deployment.automountServiceAccountToken` | bool | equality |

### 8.3 Linked Match Filtering

**Problem:** Policies with multiple fields from repeated sub-objects (e.g., containers) must ensure matches come from the SAME sub-object.

**Example:** Policy "CVE-2023-1234 AND component nginx" should match only if both CVE and component name match within the same container's image scan, not across different containers.

**Algorithm:**
1. Group policy fields by common parent path
2. Within each linked group, iterate over parent sub-objects
3. For each sub-object, evaluate all fields in the group
4. A linked group matches if ANY sub-object satisfies ALL fields

```
function evaluate_linked_group(group, parent_objects):
  for obj in parent_objects:
    if all fields in group match against obj:
      return true  // found a sub-object where all fields match
  return false
```

### 8.4 Augmented Objects

Before policy evaluation, objects are augmented with computed/derived fields:

| Augmentation | Computation |
|-------------|-------------|
| `imageAge` | `now() - image.metadata.created` |
| `unscannable` | `image.scan == null AND image.notes contains MISSING_SCAN_DATA` |
| `permissionLevel` | RBAC evaluation of service account across all bindings |
| `processBaseline` | Lookup against locked baseline for deployment+container |
| `networkBaseline` | Lookup against locked network baseline for deployment |
| `writableVolume` | Check volume mount flags |

### 8.5 Evaluation Contexts

| Context | Data Available | Sub-Types |
|---------|---------------|-----------|
| **BUILD** | Image only | Image scan, Dockerfile |
| **DEPLOY** | Deployment + Image | Full deployment spec + enriched images |
| **RUNTIME** | Deployment + Event | Process: ProcessIndicator |
| | | Network: NetworkFlow |
| | | File: FileAccess event |
| | | Audit: K8s audit log event |
| | | KubeEvent: K8s API operation |

---

## 9. Storage Model

### 9.1 Schema Generation from Protobuf

StackRox generates PostgreSQL schemas from Protocol Buffer definitions using a reflection-based walker:

```
Proto Definition (.proto)
  → Go struct tags (@gotags: sql:"pk,type(uuid)" search:"Field Name")
  → walker.Walk() (Go reflection)
  → Schema object (fields, children, references, indexes)
  → GORM model struct
  → PostgreSQL CREATE TABLE
```

### 9.2 Walker Pattern

The walker reflects on generated Go protobuf structs and builds Schema objects:

**Input:** `reflect.Type` of protobuf message (e.g., `*storage.Deployment`)

**Process:**
1. Walk each struct field recursively
2. Parse struct tags:
   - `sql:"pk"` → primary key
   - `sql:"fk(Table:col)"` → foreign key
   - `sql:"type(uuid)"` → column type override
   - `sql:"index=btree"` → create index
   - `sql:"unique"` → unique constraint
   - `search:"Field Name"` → searchable field
   - `search:"Field Name,hidden"` → indexed but not shown in autocomplete
   - `policy:"Policy Field Name"` → available as policy criterion
   - `scrub:"always"` → redacted in logs/API
3. Handle repeated fields → create child tables with parent PK + `idx` column
4. Handle maps → `jsonb` columns
5. Auto-generate `serialized bytea` column for full proto storage

**Output:** `walker.Schema` with `Fields[]`, `Children[]`, `References[]`

### 9.3 Tag System

| Tag | Purpose | Example |
|-----|---------|---------|
| `sql:"pk"` | Primary key | `sql:"pk,type(uuid)"` |
| `sql:"fk(T:c)"` | Foreign key to table T, column c | `sql:"fk(Cluster:id)"` |
| `sql:"type(X)"` | PostgreSQL column type | `uuid`, `varchar[]`, `jsonb`, `timestamptz`, `cidr` |
| `sql:"index=btree"` | B-tree index | Performance for range queries |
| `sql:"index=hash"` | Hash index (deprecated) | Equality-only queries |
| `sql:"index=gin"` | GIN index | Arrays, JSONB |
| `sql:"index=brin"` | BRIN index | Time-series data |
| `sql:"unique"` | Unique constraint | Names, identifiers |
| `search:"Label"` | Search field label | User-facing search name |
| `search:"Label,hidden"` | Hidden search field | Indexed but not in autocomplete |
| `search:"Label,store"` | Stored search field | Value stored in search index |
| `policy:"Label"` | Policy criterion | Available in policy builder |
| `scrub:"always"` | Secret scrubbing | Redacted in logs/responses |

### 9.4 PostgreSQL Data Types

| Proto Type | SQL Type | Go GORM Type |
|-----------|----------|-------------|
| `string` | `varchar` | `string` |
| `string` with `type(uuid)` | `uuid` | `string` |
| `bool` | `bool` | `bool` |
| `int32`, `uint32` | `integer` | `int32` |
| `int64`, `uint64` | `bigint` | `int64` |
| `float` | `numeric` | `float32` |
| `double` | `numeric` | `float64` |
| `bytes` | `bytea` | `[]byte` |
| `Timestamp` | `timestamp` | `*time.Time` |
| `Timestamp` with `type(timestamptz)` | `timestamptz` | `*time.Time` |
| `repeated string` | `text[]` | `*pq.StringArray` |
| `repeated enum` | `int[]` | `*pq.Int32Array` |
| `map<string,string>` | `jsonb` | `[]byte` |
| `message` (nested) | child table | separate GORM model |

### 9.5 Transaction Model

**Nested Transaction Support:**

```
outer tx (created at API boundary)
  │
  ├── Store A: sees tx from context → inner mode (commit/rollback are NOOPs)
  ├── Store B: sees tx from context → inner mode
  └── Store C: sees tx from context → inner mode
  │
  outer tx.Commit() → actual commit
  outer tx.Rollback() → actual rollback
```

**Transaction modes:**
1. `original`: Transaction created and used within a single store. Full commit/rollback.
2. `outer`: Transaction created outside stores, passed via `context.Context`. Commit/rollback controlled by creator.
3. `inner`: Store operation detects existing transaction in context. Commit/rollback are NOOPs.

**Context propagation:**
```go
tx, err := db.Begin(ctx)
ctx = postgres.ContextWithTx(ctx, tx)
// All store operations in this context share the transaction
storeA.Upsert(ctx, obj1)
storeB.Upsert(ctx, obj2)
tx.Commit(ctx)  // commits both operations atomically
```

**Error handling:** Commit/rollback use `context.WithoutCancel()` to ensure completion even if parent context is cancelled.

### 9.6 Migration System

**Bootstrapping a migration:**
```bash
DESCRIPTION="add_column_x_to_deployments" make bootstrap_migration
```

This creates `migrator/migrations/m_{N}_to_m_{N+1}_{desc}/migration_impl.go`.

**Rules:**
1. Migrations must be backward-compatible (add columns, not remove them)
2. Old code ignores new columns; new code reads new columns with fallback to old
3. Do not use feature flags in migrations
4. Do not depend on mutable code (use frozen schemas)
5. Column removal only when `MinimumSupportedDBVersionSeqNum` advances past the migration that added the replacement

### 9.7 Error Translation

PostgreSQL error codes are translated to StackRox error taxonomy:

| PG Code | PG Error | StackRox Error |
|---------|----------|---------------|
| 23505 | unique_violation | `errox.AlreadyExists` |
| 23503 | foreign_key_violation ("not present in table") | `errox.ReferencedObjectNotFound` |
| 23503 | foreign_key_violation (other) | `errox.ReferencedByAnotherObject` |
| 08xxx | connection exceptions | Transient (retryable) |
| 40xxx | transaction rollback | Transient (retryable) |
| 55xxx | lock_not_available | Transient (retryable) |
| 57xxx | operator intervention | Transient (retryable) |
| 58xxx | system errors | Transient (retryable) |

**Retry configuration:**
- Timeout: `env.PostgresQueryRetryTimeout`
- Interval: `env.PostgresQueryRetryInterval`
- Disabled: `env.PostgresDisableQueryRetries`

---

## 10. Search Specification

### 10.1 Query Language Syntax

StackRox provides a structured search query language used across UI, API, and CLI:

**Basic syntax:** `<FieldLabel>:<Value>`

**Examples:**
```
Deployment:nginx
Cluster:production+Namespace:default
CVE:CVE-2023-1234
Severity:CRITICAL_VULNERABILITY_SEVERITY
Image:quay.io/stackrox*
Risk Score:>=7.0
```

**Operators:**
- Implicit substring match: `Deployment:nginx` matches "nginx-deployment"
- Exact match: `Deployment:"nginx"` (quoted)
- Regex: `Deployment:r/nginx-.*`
- Negation: `Deployment:!nginx`
- Numeric comparison: `CVSS:>=9.0`, `Risk Score:>5`
- Boolean: `Privileged:true`
- Conjunction: `+` (AND between different fields)
- Disjunction: `,` (OR within same field)

**Pagination:**
```
query: "Cluster:prod"
pagination: { limit: 50, offset: 0, sortOption: { field: "Risk Score", reversed: true } }
```

### 10.2 Query to PostgreSQL Translation

**Pipeline:**
```
Search Query String
  → Parser → ParsedQuery { field_label: value, ... }
  → Query Builder → search.Query proto
  → postgres.Query() → SQL WHERE clause
  → Join Discovery (BFS on schema graph)
  → SAC Filter Injection
  → Execute query
```

**Join Discovery:**
The search system maintains a graph of table relationships (foreign keys). When a query references fields from multiple tables, BFS finds the shortest join path:

```
Example: Query "Deployment:nginx AND CVE:CVE-2023-1234"
  - "Deployment" field → deployments table
  - "CVE" field → image_cves_v2 table
  - Join path: deployments → deployments_containers → images_v2 → image_cves_v2
```

**SAC Filter Injection:**
After building the SQL WHERE clause, SAC filters are AND-ed:
```sql
SELECT d.* FROM deployments d
  JOIN ... ON ...
  WHERE d.name ILIKE '%nginx%'
    AND ic.cve = 'CVE-2023-1234'
    AND (d.cluster_id = 'c1' AND d.namespace IN ('ns-a', 'ns-b'))  -- SAC filter
  ORDER BY d.risk_score DESC
  LIMIT 50
```

### 10.3 Two-Phase Pagination

For queries involving joins and aggregation:

**Phase 1:** Execute query with LIMIT/OFFSET to get primary key IDs
**Phase 2:** Fetch full objects by ID (avoids serialization of full protos during sorting)

```sql
-- Phase 1: Get IDs
SELECT d.id FROM deployments d WHERE ... ORDER BY d.risk_score DESC LIMIT 50 OFFSET 0;

-- Phase 2: Get full objects
SELECT d.serialized FROM deployments d WHERE d.id IN ($1, $2, ..., $50);
```

### 10.4 Search Categories

Each searchable entity has a search category that determines which table(s) are queried:

| Category | Primary Table | Related Tables |
|----------|--------------|---------------|
| DEPLOYMENTS | deployments | deployments_containers, images_v2 |
| IMAGES | images_v2 | image_cves_v2, image_components_v2 |
| ALERTS | alerts | policies, deployments |
| POLICIES | policies | - |
| CLUSTERS | clusters | cluster_health_statuses |
| NODES | nodes | node_cves |
| NAMESPACES | namespaces | - |
| SECRETS | secrets | - |
| NETWORK_POLICIES | network_policies | - |
| PROCESS_INDICATORS | process_indicators | - |
| ROLES | k8s_roles | - |
| ROLE_BINDINGS | k8s_role_bindings | - |
| SERVICE_ACCOUNTS | service_accounts | - |
| IMAGE_VULNERABILITIES | image_cves_v2 | images_v2 |
| NODE_VULNERABILITIES | node_cves | nodes |
| COMPLIANCE_RESULTS | compliance_operator_check_results_v2 | - |

---

## 11. Scanner V4 Specification

### 11.1 Architecture

Scanner V4 is built on ClairCore and provides two main services:

**Indexer Service:** Extracts package metadata from container images
**Matcher Service:** Matches packages against vulnerability databases and enriches CVE data

### 11.2 Indexer: Layer Analysis

**Ecosystem Scanners:**

| Scanner | Target | Detection Method |
|---------|--------|-----------------|
| Alpine | APK packages | `/lib/apk/db/installed` |
| Debian | dpkg packages | `/var/lib/dpkg/status` |
| RPM | RPM packages | `/var/lib/rpm/Packages` or `/var/lib/rpm/rpmdb.sqlite` |
| Go | Go binaries | Binary version info via `debug/buildinfo` |
| Java | JAR/WAR/EAR | `META-INF/MANIFEST.MF`, `pom.properties` |
| Python | pip packages | `site-packages/*.dist-info/METADATA` |
| Node.js | npm packages | `node_modules/*/package.json` |
| Ruby | gems | `gems/*/gemspec` |

**OS Detection:**
```
Priority order:
  1. /etc/os-release (NAME, VERSION_ID)
  2. /etc/redhat-release
  3. /etc/alpine-release
  4. /etc/debian_version
  5. /usr/lib/os-release
```

**Index Report Storage:**
```sql
-- ClairCore schema
CREATE TABLE manifest (
  hash    TEXT PRIMARY KEY,
  manifest JSONB           -- Full claircore.IndexReport
);

-- StackRox extension
CREATE TABLE manifest_metadata (
  hash           TEXT PRIMARY KEY,
  expiry_time    TIMESTAMP,     -- Random 7-30 days
  scanner_version TEXT
);
```

**Cache Behavior:**
- On CreateIndexReport: check if manifest exists and scanner version matches
- If cached and not expired: return cached IndexReport
- If expired or version mismatch: re-index
- Expiry is randomized (7-30 days) to prevent thundering herd

### 11.3 Matcher: Vulnerability Matching

**Matchers (13 ecosystems):**

| Matcher | Vulnerability Source |
|---------|-------------------|
| Alpine | Alpine Security Database |
| AWS | Amazon Linux Security Advisories |
| Debian | Debian Security Tracker |
| Oracle | Oracle Linux Security Advisories |
| Photon | VMware Photon OS Advisories |
| Python | OSV (Open Source Vulnerabilities) |
| RHEL | Red Hat VEX data |
| Ruby | OSV |
| SUSE | SUSE Security Updates |
| Ubuntu | Ubuntu CVE Tracker |
| Node.js | OSV |
| Go | OSV |
| Java | OSV |

**Matching Algorithm:**
```
for each package in IndexReport.packages:
  matcher = select_matcher(package.ecosystem, distribution)
  vulnerabilities = matcher.query(
    package_name = package.name,
    package_version = package.version,
    distribution = distribution
  )
  // Version comparison is ecosystem-specific:
  //   RPM: rpmvercmp
  //   Debian: dpkg version compare
  //   Semver: semantic version compare
  //   Alpine: apk version compare
```

### 11.4 Enrichment

**Enricher Pipeline:**

| Enricher | Source | Data Provided |
|----------|-------|--------------|
| NVD | NIST NVD API / bundled data | CVSS v2/v3.0/v3.1 scores, descriptions, references |
| CSAF | Red Hat CSAF advisories | Red Hat CVSS scores, fixed-in versions, severity overrides |
| EPSS | First.org EPSS | Exploit prediction score (0-1), percentile |
| FixedBy | Distribution advisories | Fixed version for each vulnerability |

### 11.5 Vulnerability Bundle Format

**Bundle URL:** `https://definitions.stackrox.io/v4/vulnerability-bundles/{ROX_VERSION}/vulnerabilities.zip`

**Archive contents:**
```
vulnerabilities.zip
  ├── alpine.json.zst
  ├── aws.json.zst
  ├── debian.json.zst
  ├── oracle.json.zst
  ├── photon.json.zst
  ├── rhel-vex.json.zst
  ├── suse.json.zst
  ├── ubuntu.json.zst
  ├── osv.json.zst          (Go, Python, Ruby, Node.js, Java)
  ├── nvd.json.zst
  ├── epss.json.zst
  ├── manual.json.zst       (manual overrides)
  └── stackrox-rhel-csaf.json.zst
```

**Update frequency:** Checked every 5 minutes with `If-Modified-Since` header.

### 11.6 Deployment Modes

| Mode | Indexer | Matcher | Use Case |
|------|---------|---------|----------|
| Combined (default) | Same instance | Same instance | Single-cluster, simple |
| Split | Separate instance | Separate instance | Large scale, independent scaling |
| Secured Cluster | In secured cluster | In management cluster | Air-gapped, data sovereignty |

---

## 12. Collector Specification

### 12.1 eBPF Model

Collector uses CO-RE (Compile Once, Run Everywhere) BPF programs. Requires kernel 4.14+ with BTF support.

**Collection Method:** `CORE_BPF` (only supported method; legacy kernel module removed).

### 12.2 Monitored Syscalls

| Syscall | Purpose | Data Captured |
|---------|---------|---------------|
| `sys_execve` | Process execution | pid, ppid, uid, gid, comm, filename, args, container_id |
| `sys_connect` | Outbound TCP/UDP | src IP:port, dst IP:port, protocol, container_id |
| `sys_accept4` | Inbound TCP/UDP | src IP:port, dst IP:port, protocol, container_id |
| `sys_close` | Connection close | fd, container_id |

**BPF Map:** Ring buffer for event delivery to userspace. All events captured (no sampling in eBPF layer).

**Performance:** <10us overhead per syscall hook.

### 12.3 Event Formats

**ProcessSignal:**
```
{
  container_id:    string (64 chars, from cgroup)
  time:            Timestamp (kernel nanoseconds)
  name:            string (16 chars max, truncated)
  exec_file_path:  string (256 chars max)
  args:            string (512 chars max, truncated)
  pid:             uint32
  uid:             uint32
  gid:             uint32
  lineage_info:    [{parent_exec_path, parent_uid}]
}
```

**NetworkConnection:**
```
{
  container_id:    string
  local:           NetworkAddress (IP:port)
  remote:          NetworkAddress (IP:port)
  is_server:       bool (accept vs connect)
  close_timestamp: Timestamp (set if closed)
  socket_family:   SocketFamily (IPv4 | IPv6)
}
```

### 12.4 ConnTracker

**State machine per connection:**
```
INITIATED → ESTABLISHED → CLOSED
              │
              └─ (afterglow) → AFTERGLOW → CLOSED
                                  │
                                  └─ (reopen) → ESTABLISHED
```

**Afterglow Algorithm:**
```
on connection_close(conn):
  remove from active_set
  add to afterglow_set with timestamp = now()
  schedule check at now() + AFTERGLOW_DURATION (default 30s)

on afterglow_check(conn):
  if conn in active_set:
    return  // was reopened
  if now() - conn.afterglow_timestamp >= AFTERGLOW_DURATION:
    report as CLOSED
    remove from afterglow_set

on connection_open(conn):
  if conn in afterglow_set:
    remove from afterglow_set
    add to active_set  // "resurrection"
  else:
    add to active_set
```

**Effect:** Short-lived connections (health checks, DNS) that open and close within the afterglow window are suppressed entirely.

### 12.5 Delta Computation

```
previous_snapshot = {}
current_snapshot = get_all_active_connections()

new_connections = current_snapshot - previous_snapshot
closed_connections = previous_snapshot - current_snapshot

send(new_connections, closed_connections)
previous_snapshot = current_snapshot
```

### 12.6 Rate Limiting

- Process signals: 100 signals/second per container (env: `ROX_COLLECTOR_PROCESS_LIMIT`)
- Network connections: 1000 connections/container/interval
- Backpressure: buffer up to 10,000 signals; drop oldest if full

### 12.7 /proc/net/tcp Fallback

Every 30 seconds, Collector scrapes `/proc/net/tcp` and `/proc/net/tcp6` as a backup mechanism to catch connections missed by eBPF hooks. Connections in LISTEN, TIME_WAIT, or CLOSE_WAIT states are filtered out.

---

## 13. Risk Scoring Model

### 13.1 Scoring Algorithm

Risk scoring uses a **multiplicative** model. The overall risk score for a deployment is the product of individual multiplier scores:

```
overall_risk = product(multiplier.Score() for multiplier in multipliers)
```

If `overall_risk = 1.0`, the deployment has baseline risk. Each multiplier contributes a factor >= 1.0 (never < 1.0).

### 13.2 Deployment Risk Multipliers

Multipliers are evaluated in this order (descending importance):

| # | Multiplier | Max Score | Heading |
|---|-----------|-----------|---------|
| 1 | Policy Violations | 4.0 | "Policy Violations" |
| 2 | Process Baseline Violations | 4.0 | "Suspicious Process Executions" |
| 3 | Image Vulnerabilities | 4.0 | "Image Vulnerabilities" |
| 4 | Service Configuration | 2.0 | "Service Configuration" |
| 5 | Network Reachability | 2.0 | "Components Useful for Attackers" |
| 6 | Risky Component Count | 1.5 | "Risky Components in Image" |
| 7 | Component Count | 1.5 | "Image Component Count" |
| 8 | Image Age | 1.5 | "Image Freshness" |

**Overall score range:** Minimum 1.0 (all multipliers return nil or 1.0), theoretical maximum = 4.0 * 4.0 * 4.0 * 2.0 * 2.0 * 1.5 * 1.5 * 1.5 = 1296.0. In practice, scores rarely exceed 50.

### 13.3 Normalization Function

Individual multipliers use a normalization function:

```
NormalizeScore(rawScore, saturation, maxValue):
  normalized = min(rawScore, saturation) / saturation * maxValue
  return max(normalized, 1.0)
```

### 13.4 Policy Violations Multiplier

```
function Score(deployment):
  alerts = search_active_alerts(deployment_id)

  scorer = new_scorer()  // starting increment = saturation/5
  for alert in alerts:
    weight = severity_weight(alert.policy.severity)
    scorer.add(weight)

  return NormalizeScore(scorer.total, saturation=10, maxValue=4)

severity_weight:
  CRITICAL: 1.5
  HIGH:     1.25
  MEDIUM:   1.0
  LOW:      0.75
```

### 13.5 Process Baseline Violations Multiplier

```
function Score(deployment):
  violating_processes = evaluate_baselines(deployment)
  if len(violating_processes) == 0:
    return nil

  scorer = new_scorer()  // increment = 2.0, decay = 0.9
  for process in violating_processes:
    scorer.add_process()  // each: increment *= 0.9

  return NormalizeScore(scorer.total, saturation=10, maxValue=4)
```

**Diminishing returns:** 1st violation = +2.0, 2nd = +1.8, 3rd = +1.62, etc.

### 13.6 Image Vulnerabilities Multiplier

```
function Score(image):
  sum = 0
  for cve in image.scan.cves:
    sum += cvss_to_weight(cve.cvss)
    // weight function maps CVSS score to contribution

  return NormalizeScore(sum, saturation=vulnSaturation, maxValue=vulnMaxScore)
```

### 13.7 Image Risk Scoring

Images have their own risk score (separate from deployment risk):

```
image_risk = product(
  ImageVulnerabilities.Score(image),
  ImageAge.Score(image),
  ImageComponentCount.Score(image)
)
```

### 13.8 Node Risk Scoring

```
node_risk = NodeVulnerabilities.Score(node)
```

---

## 14. Notification and Enforcement

### 14.1 Alert Routing

When alerts are generated, they are routed to notifiers configured on the triggering policy:

```
for alert in new_alerts:
  for notifier_id in alert.policy.notifiers:
    notifier = registry.Get(notifier_id)
    if notifier == nil:
      log.Warn("notifier not found")
      continue
    if !label_filter_matches(notifier, alert):
      continue
    async:
      notifier.AlertNotify(ctx, alert)  // 3 retries, 10s timeout
```

### 14.2 Notifier Types

| Type | Transport | Payload |
|------|-----------|---------|
| Slack | HTTPS webhook | Formatted message with alert details, policy info, MITRE mapping |
| Jira | REST API | Issue creation with alert fields |
| PagerDuty | Events API v2 | Incident trigger |
| Email | SMTP | Templated email |
| Generic Webhook | HTTPS POST | JSON alert payload |
| Splunk | HTTP Event Collector | JSON event |
| AWS Security Hub | AWS API | Security finding |
| Google Cloud SCC | Google API | Security finding |
| Syslog | TCP/UDP | CEF or JSON format |
| Microsoft Teams | HTTPS webhook | Adaptive card |

### 14.3 Alert Formatting

Alerts are formatted using Go templates. The template context includes:

```
{
  alert:         storage.Alert
  policy:        storage.Policy
  violations:    []storage.Violation
  deployment:    storage.Deployment (or image/resource/node)
  alertLink:     string (URL to Central UI)
  mitreVectors:  []MitreAttackVector (resolved from MITRE store)
}
```

### 14.4 Enforcement Actions

| Action | Executor | Mechanism | Reversible |
|--------|----------|-----------|------------|
| SCALE_TO_ZERO | Sensor | `kubectl scale --replicas=0` | Yes (original replicas saved in annotation) |
| KILL_POD | Sensor | `kubectl delete pod --selector=<labels>` | No (pods recreated by controller, but may fail admission) |
| FAIL_BUILD | Central | roxctl exits with non-zero status | N/A |
| FAIL_KUBE_REQUEST | Admission Controller | Webhook returns `Allowed: false` | N/A |
| FAIL_DEPLOYMENT_CREATE | Sensor | K8s API rejection | N/A |
| FAIL_DEPLOYMENT_UPDATE | Sensor | K8s API rejection | N/A |

### 14.5 Break-Glass Bypass

Deployments can bypass admission enforcement using annotations:

```yaml
metadata:
  annotations:
    # Bypass all enforcement (still generates alerts)
    admission.stackrox.io/break-glass: "ticket-1234"

    # Disable specific policies
    admission.stackrox.io/policy-ids-disabled: "policy-id-1,policy-id-2"
```

When break-glass is used:
1. Admission webhook allows the deployment
2. Alert is still generated with bypass context
3. Audit trail preserved for security review

---

## 15. Compliance Model

### 15.1 Architecture

StackRox compliance assessment uses the OpenShift Compliance Operator as the execution engine:

```
Compliance Operator (in cluster)
  │  Runs OpenSCAP scans
  │  Produces ComplianceCheckResults CRDs
  │
  ▼
Sensor (watches Compliance Operator CRDs)
  │  Converts to storage.ComplianceOperatorCheckResultV2
  │
  ▼
Central
  │  Stores results in PostgreSQL
  │  Aggregates across clusters
  │  Provides reporting APIs
```

### 15.2 Standards

| Standard | Product Types | Source |
|----------|--------------|-------|
| CIS Kubernetes Benchmark | ocp4, rhcos4 | Compliance Operator profiles |
| PCI-DSS | ocp4 | Compliance Operator profiles |
| HIPAA | ocp4 | Compliance Operator profiles |
| NIST 800-53 | ocp4 | Compliance Operator profiles |
| NERC-CIP | ocp4 | Compliance Operator profiles |
| FedRAMP | ocp4 | Compliance Operator profiles |

### 15.3 Check Results

```
CheckStatus enum:
  PASS            // Check passed
  FAIL            // Check failed
  ERROR           // Check could not be evaluated
  INFO            // Informational only
  MANUAL          // Requires manual verification
  NOT_APPLICABLE  // Check does not apply to this environment
  INCONSISTENT    // Results differ across nodes
```

### 15.4 Scan Configuration

```
ComplianceScanConfigurationV2 {
  id:                     string
  scan_name:              string
  scan_config:            BaseComplianceScanConfigurationSettings
  clusters:               []ClusterScanStatus
  created_time:           Timestamp
  last_updated_time:      Timestamp
  modified_by:            SlimUser
  description:            string
}
```

### 15.5 Result Aggregation

Results are aggregated at multiple levels:
1. **Check level:** Individual check result per node/cluster
2. **Profile level:** Percentage of passing checks per profile
3. **Cluster level:** Overall compliance score per cluster
4. **Standard level:** Cross-cluster compliance percentage per standard

---

## 16. Certificate and Identity Model

### 16.1 mTLS Hierarchy

```
StackRox CA (self-signed, generated at install)
  │
  ├── Central certificate
  │     CN: central.stackrox
  │     SANs: central.stackrox, central.stackrox.svc, ...
  │
  ├── Scanner V4 certificate
  │     CN: scanner-v4.stackrox
  │
  ├── Sensor certificate (per cluster)
  │     CN: SENSOR_SERVICE
  │     SANs: sensor.stackrox.svc
  │
  ├── Collector certificate (per cluster)
  │     CN: COLLECTOR_SERVICE
  │
  └── Admission Controller certificate (per cluster)
       CN: ADMISSION_CONTROL_SERVICE
```

### 16.2 CA Lifecycle

**Primary CA:** Generated during initial installation. Used to sign all service certificates.

**Secondary CA (for rotation):**
1. New CA generated
2. Both CAs distributed to all components (cert bundle)
3. New certificates signed by new CA
4. Old CA retired after all certificates rotated

### 16.3 Init Bundles

Init bundles are pre-generated certificate packages used to register new clusters:

```
Init Bundle {
  ca_cert:        PEM (CA certificate)
  sensor_cert:    PEM (Sensor certificate)
  sensor_key:     PEM (Sensor private key)
  collector_cert: PEM (Collector certificate)
  collector_key:  PEM (Collector private key)
  admission_control_cert: PEM
  admission_control_key:  PEM
}
```

Created via `roxctl init-bundle create` or Central API. Applied to the secured cluster as Kubernetes Secrets.

### 16.4 Service Identity Types

| Identity | mTLS CN | Purpose |
|----------|---------|---------|
| CENTRAL_SERVICE | central.stackrox | Central API server |
| SENSOR_SERVICE | sensor.stackrox | Sensor agent |
| COLLECTOR_SERVICE | collector.stackrox | Collector daemon |
| ADMISSION_CONTROL_SERVICE | admission-control.stackrox | Admission webhook |
| SCANNER_SERVICE | scanner-v4.stackrox | Scanner V4 |
| SCANNER_DB_SERVICE | scanner-v4-db.stackrox | Scanner database |
| CENTRAL_DB_SERVICE | central-db.stackrox | Central database |

### 16.5 Certificate Expiry

- Service certificates: 1 year default, auto-rotated
- CA certificate: 5 years default
- Sensor monitors certificate expiry and triggers refresh
- Central monitors all component certificate status

---

## 17. Failure Model and Recovery

### 17.1 Sensor Disconnection

| Capability | Online | Offline |
|-----------|--------|---------|
| K8s resource monitoring | Yes | Yes (informers continue) |
| Local store updates | Yes | Yes |
| Admission control | Yes (fresh policies) | Yes (cached policies, eventual consistency) |
| Runtime event buffering | Yes (streaming) | Yes (bounded queues, drops when full) |
| Enforcement actions | Yes | Yes (K8s API direct) |
| Image enrichment | Yes (from Central) | No (stale scan data) |
| Policy updates | Yes (Central sync) | No (uses last-known policies) |
| Alert transmission | Yes (streaming) | No (buffered, sent on reconnect) |

**Queue capacities in offline mode:**
- Process indicators: 10,000
- Network flows: 10,000
- File access events: 1,000

**Reconnection behavior:**
1. Exponential backoff retry (5s initial, 5min max)
2. On reconnect: SensorHello with RECONNECT state
3. Deduper state exchange (send only changed resources)
4. Flush buffered events within expiration time
5. Receive fresh policies and baselines from Central

### 17.2 Scanner Unavailability

| Scenario | Behavior |
|----------|----------|
| Scanner V4 unreachable | Image enrichment fails; `MISSING_SCAN_DATA` note added to image |
| Scanner DB unreachable | Scanner returns error; Central retries |
| Vulnerability bundle stale | Matcher continues with last-known data; metrics track staleness |
| Registry auth failure | Integration marked unhealthy after 3 consecutive failures |
| Registry timeout | Retry with exponential backoff (scanner client) |
| Registry rate limit | Respect 429 responses, wait and retry |

### 17.3 Database Connection Loss

| Scenario | Behavior |
|----------|----------|
| PostgreSQL connection lost | Retry transaction up to 3 times (transient error detection) |
| Deadlock detected | Retry with random jitter |
| Unique constraint violation | Log and skip (duplicate key, idempotent) |
| Connection pool exhausted | Queue requests until connection available |
| PostgreSQL process restart | pgx reconnects automatically via pool |

### 17.4 Component Health States

Health is determined by Central based on connection state and heartbeat data:

```
Component Health State Machine:

  UNINITIALIZED
       │
       │ First connection
       ▼
    HEALTHY ◄──── heartbeat received within threshold
       │
       │ heartbeat missed (threshold: configurable)
       ▼
    DEGRADED
       │
       │ multiple heartbeats missed
       ▼
    UNHEALTHY
       │
       │ component removed / not deployed
       ▼
    UNAVAILABLE (Collector only)
```

**Health thresholds:**
- Sensor: heartbeat within gRPC stream (connection loss = immediate UNHEALTHY)
- Collector: reported by Sensor; threshold-based degradation
- Admission Controller: reported by Sensor; threshold-based degradation

### 17.5 Data Loss Scenarios

| Scenario | Data at Risk | Mitigation |
|----------|-------------|------------|
| Sensor crash | Buffered events not yet sent | Bounded queues; events re-created on K8s reconciliation; runtime events lost |
| Central crash | In-flight transactions | PostgreSQL WAL ensures durability; uncommitted transactions rolled back |
| Collector crash | BPF ring buffer contents | Events since last flush lost (5s window max) |
| PostgreSQL crash | Committed transactions | WAL + checkpoints; fsync ensures durability |
| Scanner crash | In-progress scans | Central retries; manifest cache preserved in Scanner DB |

---

## 18. Configuration Reference

### 18.1 Central Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `ROX_CENTRAL_DB` | (from config) | PostgreSQL connection string |
| `ROX_PLAINTEXT_ENDPOINTS` | (none) | Additional plaintext HTTP endpoints |
| `ROX_MTLS_CERT_FILE` | `/run/secrets/stackrox.io/certs/cert.pem` | TLS certificate |
| `ROX_MTLS_KEY_FILE` | `/run/secrets/stackrox.io/certs/key.pem` | TLS private key |
| `ROX_MTLS_CA_FILE` | `/run/secrets/stackrox.io/certs/ca.pem` | CA certificate |
| `ROX_IMAGE_FLAVOR` | `rhacs` | Image flavor for defaults |
| `ROX_NETPOL_FIELDS` | (off) | Enable network policy fields |

### 18.2 Sensor Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `ROX_CENTRAL_ENDPOINT` | `central.stackrox:443` | Central gRPC endpoint |
| `ROX_ADVERTISED_ENDPOINT` | `sensor.stackrox:443` | Sensor endpoint for Collector/AC |
| `CLUSTER_NAME` | (required) | Cluster identifier |
| `ROX_CONNECTION_RETRY_INITIAL_INTERVAL` | `5s` | Initial reconnect backoff |
| `ROX_CONNECTION_RETRY_MAX_INTERVAL` | `5m` | Maximum reconnect backoff |
| `ROX_PROCESS_FILTER_LEVEL` | `default` | Process filter aggressiveness (`aggressive`/`default`/`minimal`) |
| `ROX_NETWORK_FLOW_BUFFER_SIZE` | `10000` | Network flow queue capacity |
| `ROX_SENSOR_INTERNAL_PUBSUB` | `false` | Enable internal PubSub system |
| `ROX_LOCAL_IMAGE_SCANNING_ENABLED` | `false` | Enable local Scanner V4 |
| `ROX_PROCESSES_LISTENING_ON_PORT` | `true` | Track processes listening on ports |

### 18.3 Collector Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `ROX_COLLECTOR_PROCESS_LIMIT` | `100` | Max process signals/sec/container |
| `GRPC_SERVER` | `sensor.stackrox:443` | Sensor gRPC endpoint |
| `COLLECTION_METHOD` | `CORE_BPF` | eBPF collection method |

### 18.4 Scanner V4 Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `SCANNER_V4_HTTP_LISTEN_ADDR` | `127.0.0.1:9443` | HTTP health/metrics endpoint |
| `SCANNER_V4_GRPC_LISTEN_ADDR` | `127.0.0.1:8443` | gRPC API endpoint |
| `SCANNER_V4_INDEXER_ENABLE` | `true` | Enable indexer mode |
| `SCANNER_V4_MATCHER_ENABLE` | `true` | Enable matcher mode |
| `SCANNER_V4_INDEXER_DATABASE_CONN_STRING` | (required) | Indexer PostgreSQL DSN |
| `SCANNER_V4_MATCHER_DATABASE_CONN_STRING` | (required) | Matcher PostgreSQL DSN |
| `SCANNER_V4_MATCHER_VULNERABILITIES_URL` | `https://definitions.stackrox.io/...` | Vulnerability bundle URL |
| `SCANNER_V4_MATCHER_READINESS` | `vulnerability` | Readiness strategy (`database`/`vulnerability`) |
| `SCANNER_V4_MTLS_CERTS_DIR` | `/run/secrets/stackrox.io/certs` | TLS certificates directory |
| `SCANNER_V4_LOG_LEVEL` | `info` | Log level |
| `SCANNER_V4_INDEXER_GET_LAYER_TIMEOUT` | `1m` | Layer download timeout |

### 18.5 PostgreSQL Configuration

| Variable | Default | Purpose |
|----------|---------|---------|
| `ROX_POSTGRES_DEFAULT_STATEMENT_TIMEOUT` | (pgx default) | Default query timeout |
| `ROX_POSTGRES_QUERY_RETRY_TIMEOUT` | (configurable) | Retry timeout for transient errors |
| `ROX_POSTGRES_QUERY_RETRY_INTERVAL` | (configurable) | Retry interval |
| `ROX_POSTGRES_DISABLE_QUERY_RETRIES` | `false` | Disable automatic retries |
| `ROX_POSTGRES_KEEP_TEST_DB` | `false` | Keep test databases (development) |

### 18.6 Feature Flags

Feature flags are environment variables with `ROX_` prefix that enable/disable functionality:

| Flag | Default | Purpose |
|------|---------|---------|
| `ROX_SCANNER_V4_REINDEX` | enabled | Automatic re-indexing of expired manifests |
| `ROX_SCANNER_V4_RED_HAT_CSAF` | enabled | Red Hat CSAF enricher |
| `ROX_SCANNER_V4_MAVEN_SEARCH` | disabled | Maven Central API for Java |
| `ROX_NETWORK_DETECTION_BASELINE_SIMULATION` | disabled | Network baseline simulation mode |
| `ROX_LABEL_BASED_POLICY_SCOPING` | disabled | Label-based policy scoping |

### 18.7 Helm Chart Configuration Surface

The primary configuration surface for deployment is the Helm chart. Key values:

**Central Services (`stackrox-central-services`):**
```yaml
central:
  image:
    registry: registry.redhat.io
    name: advanced-cluster-security/rhacs-main-rhel8
    tag: <version>
  resources:
    requests: { memory: 4Gi, cpu: 1500m }
    limits: { memory: 8Gi, cpu: 4000m }
  exposure:
    loadBalancer: { enabled: false, port: 443 }
    nodePort: { enabled: false }
    route: { enabled: false }  # OpenShift only
  db:
    source: # PostgreSQL connection config
    persistence: { persistentVolumeClaim: { claimName: central-db } }

scanner:
  replicas: 3
  autoscaling: { disable: false, minReplicas: 2, maxReplicas: 5 }

scannerV4:
  indexer:
    replicas: 1
  matcher:
    replicas: 1
  db:
    persistence: { persistentVolumeClaim: { claimName: scanner-v4-db } }
```

**Secured Cluster Services (`stackrox-secured-cluster-services`):**
```yaml
clusterName: <required>
centralEndpoint: central.stackrox:443

sensor:
  resources:
    requests: { memory: 1Gi, cpu: 1000m }

collector:
  collectionMethod: CORE_BPF

admissionControl:
  listenOnCreates: true
  listenOnUpdates: false
  listenOnEvents: true
  contactImageScanners: DoNotScanInline
  timeout: 20  # seconds
```

---

## Appendix A: Entity Relationship Summary

```
Cluster (1)
  ├──► (N) Deployment
  │      ├──► (N) Container
  │      │      └──► (1) ImageV2
  │      │             ├──► (N) ImageComponentV2
  │      │             │      └──► (N) ImageCVEV2
  │      │             │             └──► (1) CVEInfo
  │      │             └──► (N) ImageSignature
  │      ├──► (N) Alert
  │      │      └──► (1) Policy
  │      ├──► (1) ProcessBaseline
  │      ├──► (1) NetworkBaseline
  │      └──► (1) Risk
  ├──► (N) Node
  │      └──► (N) NodeCVE
  ├──► (N) NetworkFlow (partitioned by cluster)
  ├──► (N) ProcessIndicator
  ├──► (N) K8sRole / K8sRoleBinding
  ├──► (N) Secret
  ├──► (N) ServiceAccount
  ├──► (N) NetworkPolicy
  ├──► (N) Namespace
  └──► (1) ClusterHealthStatus

Policy (1)
  ├──► (N) PolicySection
  │      └──► (N) PolicyGroup
  │             └──► (N) PolicyValue
  ├──► (N) Exclusion
  ├──► (N) Scope
  └──► (N) MitreAttackVector

Role (1)
  ├──► (1) PermissionSet
  └──► (1) SimpleAccessScope

ComplianceOperatorProfileV2 (1)
  └──► (N) ComplianceOperatorRuleV2
         └──► (N) ComplianceOperatorCheckResultV2
```

## Appendix B: API Service Summary

### v1 Services (60 services, ~200 RPC methods)

| Service | Primary Resource | Key Methods |
|---------|-----------------|-------------|
| AlertService | Alert | GetAlert, ListAlerts, ResolveAlert, DeleteAlerts |
| PolicyService | Policy | GetPolicy, ListPolicies, PostPolicy, PutPolicy, DeletePolicy, DryRunPolicy, ImportPolicies |
| DeploymentService | Deployment | GetDeployment, ListDeployments, ExportDeployments |
| ImageService | Image | GetImage, ScanImage, ListImages, ExportImages |
| ClustersService | Cluster | GetCluster, PostCluster, PutCluster, DeleteCluster |
| NodeService | Node | GetNode, ListNodes |
| CVEService | CVE | GetCVE, SuppressCVE, UnsuppressCVE |
| RoleService | Role | GetRole, ListRoles, CreateRole, UpdateRole, DeleteRole |
| AuthService | Auth | GetAuthStatus |
| NetworkGraphService | NetworkGraph | GetNetworkGraph, GetExternalNetworkEntities |
| NetworkPolicyService | NetworkPolicy | GetNetworkPolicies, ApplyNetworkPolicy |
| ProcessService | ProcessIndicator | GetProcesses |
| SecretService | Secret | GetSecret, ListSecrets |
| SearchService | (global) | Search, Autocomplete |
| MetadataService | (system) | GetMetadata |
| ImageIntegrationService | ImageIntegration | GetImageIntegrations, PostImageIntegration |
| NotifierService | Notifier | GetNotifiers, PostNotifier, TestNotifier |
| BackupService | Backup | GetBackup, TriggerBackup |

### v2 Services (10 services)

| Service | Purpose |
|---------|---------|
| ComplianceIntegrationService | Compliance Operator integration management |
| ComplianceResultsService | Compliance scan results |
| ComplianceProfileService | Compliance profile management |
| ComplianceScanConfigurationService | Scan configuration |
| VulnExceptionService | Vulnerability exception management |
| ReportServiceV2 | Enhanced reporting |
| BaseImageService | Base image management |

### Internal Services

| Service | Protocol | Purpose |
|---------|----------|---------|
| SensorService.Communicate | Bidirectional gRPC stream | Central <-> Sensor communication |
| Indexer.CreateIndexReport | gRPC | Scanner V4 indexing |
| Matcher.GetVulnerabilities | gRPC | Scanner V4 matching |
| AdmissionControlManagementService | gRPC | Sensor -> Admission Controller |
| SignalService.PushSignals | gRPC stream | Collector -> Sensor |

## Appendix C: Metrics Reference

### Central Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `central_postgres_query_errors` | Counter | query, error | PostgreSQL query errors |
| `rox_alerts_generated_total` | Counter | lifecycle_stage, severity | Alerts created |
| `rox_detection_duration_seconds` | Histogram | stage | Detection time by lifecycle |
| `rox_enforcement_actions_total` | Counter | action | Enforcement actions executed |
| `rox_notifications_sent_total` | Counter | notifier_type | Notifications sent |
| `rox_networkgraph_flows_stored_total` | Counter | - | Network flows upserted |

### Sensor Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `sensor_k8s_event_count` | Counter | action, dispatcher | K8s events processed |
| `sensor_detector_deployment_processed` | Counter | - | Deployments evaluated |
| `sensor_detector_cache_hit` | Counter | - | Deduplication cache hits |
| `sensor_detector_queue_size` | Gauge | queue | Current queue depths |
| `sensor_grpc_message_count` | Counter | direction | gRPC messages sent/received |
| `sensor_network_flow_dropped_total` | Counter | - | Network flows dropped (queue full) |

### Collector Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `collector_network_connections_active` | Gauge | Active connections tracked |
| `collector_dropped_network_flows_total` | Counter | Connections dropped (rate limit) |

---

*End of Specification*
