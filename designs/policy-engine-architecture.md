# StackRox Policy Engine Architecture

## Executive Summary

The StackRox policy engine is a sophisticated, multi-layered security policy evaluation system that enforces security policies across the entire application lifecycle (Build, Deploy, Runtime). The engine uses a criteria-based approach with boolean logic to evaluate deployments, images, and runtime behavior against configurable security policies. All policy evaluation is **workload-centric**, meaning every evaluation occurs within the context of a deployment enriched with related security data.

## Table of Contents

1. [Core Architecture](#core-architecture)
2. [Policy Data Structure](#policy-data-structure)
3. [Criteria System](#criteria-system)
4. [Policy Context System](#policy-context-system)
5. [Policy Compilation and Evaluation](#policy-compilation-and-evaluation)
6. [Enforcement Mechanisms](#enforcement-mechanisms)
7. [Key Implementation Details](#key-implementation-details)

## Core Architecture

### High-Level Components

The StackRox policy engine consists of several key architectural layers:

1. **Policy Storage Layer** - Protobuf-based policy definitions stored in PostgreSQL
2. **Boolean Policy Engine** - Core evaluation logic with criteria matching
3. **Context Augmentation System** - Enriches objects with evaluation context
4. **Detection Engine** - Multi-stage detection orchestration (Build/Deploy/Runtime)
5. **Enforcement Layer** - Executes policy actions based on violations

### Key Directories

- `/proto/storage/policy.proto` - Core policy protobuf definitions
- `/central/policy/` - Policy management services and data stores
- `/pkg/booleanpolicy/` - Core policy evaluation logic
- `/pkg/booleanpolicy/evaluator/` - Policy criteria evaluation engine
- `/pkg/booleanpolicy/fieldnames/` - Definition of all policy criteria fields
- `/central/detection/` - Multi-stage detection orchestration
- `/pkg/detection/` - Core detection interfaces and compiled policies

## Policy Data Structure

### Core Policy Definition

```protobuf
message Policy {
  string id = 1;
  string name = 2;
  string description = 3;
  bool disabled = 6;
  repeated LifecycleStage lifecycle_stages = 9;  // BUILD, DEPLOY, RUNTIME
  EventSource event_source = 22;  // NOT_APPLICABLE, DEPLOYMENT_EVENT, AUDIT_LOG_EVENT
  repeated PolicySection policy_sections = 20;   // Core criteria definition
  Severity severity = 12;  // LOW, MEDIUM, HIGH, CRITICAL
  repeated EnforcementAction enforcement_actions = 13;
  repeated Scope scope = 11;  // Cluster/namespace targeting
  repeated Exclusion exclusions = 21;  // Policy exemptions
}
```

### Policy Criteria Structure

```protobuf
message PolicySection {
  string section_name = 1;
  repeated PolicyGroup policy_groups = 3;  // Individual criteria
}

message PolicyGroup {
  string field_name = 1;  // The specific criteria field
  BooleanOperator boolean_operator = 2;  // OR/AND for multiple values
  bool negate = 3;  // Invert the match logic
  repeated PolicyValue values = 4;  // Values to match against
}
```

### Boolean Logic

- **Policy Sections**: Implicit AND logic (all sections must match)
- **Policy Groups**: Configurable OR/AND operators for multiple values
- **Negation**: Most criteria support negation (some forbidden for logical reasons)

Example:
```
Policy Section: "High-Risk Configuration" 
  AND Policy Group: "Privileged Container" = true
  AND Policy Group: "CVE" = ["CVE-2021-44228", "CVE-2021-45046"] (OR logic)
  AND Policy Group: "Severity" >= "HIGH"
```

## Criteria System

### Complete Criteria Catalog (85 Fields)

The policy engine supports 85 different criteria fields organized into categories:

#### Container Security (13 fields)
- `Add Capabilities`, `Drop Capabilities`
- `Allow Privilege Escalation`, `Privileged Container`
- `Read-Only Root Filesystem`
- `AppArmor Profile`, `Seccomp Profile Type`
- `Container CPU/Memory Limit/Request`
- `Automount Service Account Token`
- `Container Name`, `Liveness/Readiness Probe Defined`

#### Image Security (21 fields)
- `CVE`, `CVSS`, `NVD CVSS`, `Severity`
- `Fixable`, `Fixed By`
- `Image Age`, `Image Scan Age`
- `Image Component`, `Image OS`
- `Image Registry`, `Image Remote`, `Image Tag`, `Image User`
- `Dockerfile Line`, `Unscanned Image`
- `Image Signature Verified By`
- `Days Since CVE Was Published/First Discovered`
- `Disallowed/Required Image Label`

#### Network & Service (9 fields)
- `Exposed Port`, `Port Exposure Method`, `Exposed Port Protocol`
- `Exposed Node Port`
- `Host Network`, `Host IPC`, `Host PID`
- `Has Ingress/Egress Network Policy`
- `Unexpected Network Flow Detected`

#### Runtime Behavior (8 fields)
- `Process Name`, `Process Arguments`, `Process Ancestor`, `Process UID`
- `Unexpected Process Executed`
- `Environment Variable`
- `Replicas`

#### Kubernetes Resources (17 fields)
- `Namespace`, `Service Account`
- `Required/Disallowed Annotation`
- `Required/Disallowed Label`
- `Minimum RBAC Permissions`
- `Kubernetes Resource`, `Kubernetes API Verb`
- `Kubernetes Resource Name`, `Kubernetes User Name`, `Kubernetes User Groups`
- `User Agent`, `Source IP Address`, `Is Impersonated User`
- `Runtime Class`

#### Storage & Volumes (9 fields)
- `Volume Type`, `Volume Source`, `Volume Destination`, `Volume Name`
- `Writable Host Mount`, `Writable Mounted Volume`
- `Mount Propagation`

### Criteria Metadata System

Each criteria field has rich metadata defining its behavior:

```go
type metadataAndQB struct {
    operatorsForbidden bool              // Single value only
    negationForbidden  bool              // Cannot be negated
    qb                 querybuilders.QueryBuilder  // Query construction logic
    valueRegex         func(*validateConfiguration) *regexp.Regexp  // Value validation
    contextFields      violationmessages.ContextQueryFields  // Related context
    eventSourceContext []storage.EventSource  // Lifecycle stage applicability
    fieldTypes         []RuntimeFieldType  // Runtime event categorization
}
```

### Value Validation

Criteria values are validated using field-specific regex patterns:
- Boolean values: `(?i:(true|false))`
- Numeric comparisons: `(<|>|<=|>=)?[[:space:]]*[[:digit:]]*\.?[[:digit:]]+`
- Severity comparisons: `(<|>|<=|>=)?[[:space:]]*(?i:UNKNOWN|LOW|MODERATE|IMPORTANT|CRITICAL)`
- Linux capabilities: Validated against known capability set
- Environment variables: Source-aware validation patterns
- IP addresses: IPv4/IPv6 validation

## Policy Context System

### Core Context Model

Every policy evaluation occurs against an **"Enhanced Deployment"** - the deployment object augmented with related contextual data:

```go
type EnhancedDeployment struct {
    Deployment             *storage.Deployment        // Base workload
    Images                 []*storage.Image           // Associated container images  
    NetworkPoliciesApplied *NetworkPoliciesApplied    // Network security context
}
```

### Context Augmentation Framework

The policy engine uses an **augmentation system** that enriches the base deployment with:

1. **Image Context**: Vulnerability data, Dockerfile analysis, component versions
2. **Runtime Context**: Process indicators, network flows, baseline deviations
3. **Security Context**: RBAC permissions, network policies, security constraints
4. **Compliance Context**: Annotations, labels, configuration compliance

### The "Augmented Object" Structure

The augmented object combines:
**Base Object** + **Augmented Fields** = **Complete Evaluation Context**

Key augmentation types:
- **Images**: Added at `Containers[i].Image` path
- **Processes**: Added at `Containers[i].ProcessIndicator` path  
- **Environment Variables**: Composite fields combining source + key + value
- **Dockerfile Lines**: Composite fields combining instruction + value
- **Network Flows**: Added at `NetworkFlow` path
- **Baseline Results**: Process and network baseline deviation flags

### Lifecycle-Specific Context

#### BUILD Time (Images)
- **Context**: Pure image data with vulnerability scans, components, Dockerfile analysis
- **Augmentations**: Component versions, Dockerfile lines, signature verification
- **Factory**: `imageEvalFactory` using `ImageMeta`

#### DEPLOY Time (Deployments)
- **Context**: Deployment spec + associated images + network policies
- **Augmentations**: All image data PLUS deployment-specific fields (volumes, security context, etc.)
- **Factory**: `deploymentEvalFactory` using `DeploymentMeta`

#### RUNTIME (Events)
- **Context**: Deployment + runtime events (processes, network flows, K8s events)
- **Augmentations**: Process indicators, network flow details, baseline deviations
- **Specialized Matchers**: Process, NetworkFlow, KubeEvent, AuditLogEvent

### Context Field Linking

The system establishes relationships between entities through several mechanisms:

#### Container-Image Linking
```go
// Images must correspond one-to-one with container specs
if len(images) != len(deployment.GetContainers()) {
    return nil, errors.Errorf("deployment %s/%s had %d containers, but got %d images",
        deployment.GetNamespace(), deployment.GetName(), 
        len(deployment.GetContainers()), len(images))
}
```

#### Process-Container Linking
```go
func findMatchingContainerIdxForProcess(deployment *storage.Deployment, 
                                       process *storage.ProcessIndicator) (int, error) {
    for i, container := range deployment.GetContainers() {
        if container.GetName() == process.GetContainerName() {
            return i, nil
        }
    }
    // Error if no match found
}
```

#### Context Field Propagation
The system automatically includes related fields during evaluation through `contextFields` mapping in field metadata.

### Key Architectural Insight

While policies are **deployment-centric**, the evaluation context can include:
- **Multiple images** (one per container)
- **Multiple processes** (runtime events)
- **Multiple network flows** (runtime events)
- **Multiple audit events** (Kubernetes API events)

The deployment serves as the **anchor point** for all related security context, enabling policies like "Deployment with high-severity CVEs AND privilege escalation AND unexpected network traffic" to be evaluated as a single coherent security assessment.

## Policy Compilation and Evaluation

### Policy Compilation Process

1. **Section to Query Translation**
```go
func sectionToQuery(section *storage.PolicySection, stage storage.LifecycleStage) (*query.Query, error) {
    fieldQueries, err := sectionToFieldQueries(section)
    contextQueries := constructRemainingContextQueries(stage, section, fieldQueries)
    fieldQueries = append(fieldQueries, contextQueries...)
    return &query.Query{FieldQueries: fieldQueries}, nil
}
```

2. **Policy Group Processing**
Each policy group is converted to field queries with validation:
- Metadata lookup for field validation
- Negation and operator restriction enforcement
- Query builder delegation for field-specific logic

3. **Compiled Policy Structure**
```go
func newCompiledPolicy(policy *storage.Policy) (CompiledPolicy, error) {
    compiled := &compiledPolicy{policy: policy}
    
    // Compile exclusions and scopes
    exclusions := make([]*compiledExclusion, 0, len(policy.GetExclusions()))
    scopes := make([]*scopecomp.CompiledScope, 0, len(policy.GetScope()))
    
    // Set up lifecycle-specific matchers
    if policies.AppliesAtRunTime(policy) {
        compiled.setRuntimeMatchers(policy)
    }
    if policies.AppliesAtDeployTime(policy) {
        compiled.setDeployTimeMatchers(policy)  
    }
    if policies.AppliesAtBuildTime(policy) {
        compiled.setBuildTimeMatchers(policy)
    }
}
```

### Query Evaluation Engine

#### Evaluator Architecture
```go
type Evaluator interface {
    Evaluate(obj pathutil.AugmentedValue) (*Result, bool)
}

type Factory struct {
    fieldToMetaPaths *pathutil.FieldToMetaPathMap
    rootType         reflect.Type
}
```

#### Evaluation Process
1. **Field Query Evaluation**: Each criteria field is evaluated independently
2. **Linked Conjunction**: All field queries must match AND be in the same object
3. **Path Filtering**: Results are filtered to ensure matches are properly linked
4. **Panic Recovery**: Evaluation errors are caught and logged as programming errors

### Policy Evaluation Flow

1. **Policy Compilation**: Policies are compiled into specialized matchers based on lifecycle stage and field types
2. **Context Construction**: Objects are augmented with relevant contextual data using `ConstructDeployment()`, `ConstructDeploymentWithProcess()`, etc.
3. **Predicate Checking**: Scope and exclusion predicates determine if policy applies to the object
4. **Field Evaluation**: Boolean policy evaluators traverse the augmented object using field paths
5. **Violation Generation**: Matching violations are collected and converted to alerts

### Context-Aware Evaluation

**Context Fields**: The engine automatically adds context fields to ensure complete evaluation:
- Container context fields for container-related criteria
- Image context fields for image-related criteria  
- Volume context fields for storage-related criteria
- Process context fields for runtime behavior criteria

## Enforcement Mechanisms

### Enforcement Actions

- `SCALE_TO_ZERO_ENFORCEMENT` - Scale deployment to zero replicas
- `UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT` - Add impossible node constraints
- `KILL_POD_ENFORCEMENT` - Terminate running pods
- `FAIL_BUILD_ENFORCEMENT` - Fail image builds
- `FAIL_KUBE_REQUEST_ENFORCEMENT` - Block kubectl operations
- `FAIL_DEPLOYMENT_CREATE_ENFORCEMENT` - Block deployment creation
- `FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT` - Block deployment updates

### Lifecycle Stage Detection

**Build Time** (`/central/detection/buildtime/`)
- Evaluates container images during CI/CD
- Focuses on vulnerability and configuration criteria
- Generates alerts but typically doesn't enforce

**Deploy Time** (`/central/detection/deploytime/`)
- Evaluates Kubernetes manifests via admission controller
- Can block deployments that violate policies
- Primary enforcement point for many security controls

**Runtime** (`/central/detection/runtime/`)
- Monitors live processes, network flows, and Kubernetes events
- Detects behavioral anomalies and runtime violations
- Can terminate processes or scale deployments

### Event Source Context

- `NOT_APPLICABLE`: Build and deploy-time policies
- `DEPLOYMENT_EVENT`: Runtime deployment events
- `AUDIT_LOG_EVENT`: Kubernetes audit log events

### Runtime Field Type Categorization

- `Process`: Process execution events
- `NetworkFlow`: Network traffic events  
- `AuditLogEvent`: Kubernetes audit events
- `KubeEvent`: Admission controller events

## Key Implementation Details

### Validation and Error Handling

**Comprehensive Validation**:
- Field name validation against known criteria
- Value format validation using regex patterns
- Operator restriction enforcement
- Negation restriction enforcement
- Context field completion

**Error Recovery**:
- Panic catching in evaluators
- Graceful degradation on evaluation errors
- Detailed error messages for policy compilation failures

### Performance Optimizations

- **Compiled Policies**: Policies are pre-compiled into efficient evaluator structures
- **Field Metadata Caching**: Singleton pattern for field metadata access
- **Path Optimization**: Efficient path traversal for nested object evaluation
- **Context Caching**: `CacheReceptacle` pattern for reusing expensive computations

### Extensibility

**Adding New Criteria**:
1. Define field name in `fieldnames/list.go`
2. Register metadata in `field_metadata.go` 
3. Implement query builder in `querybuilders/`
4. Add value validation regex if needed

### Architecture Benefits

- **Separation of Concerns**: Context augmentation is separated from policy logic
- **Type Safety**: Strong typing through Go structs ensures consistent field access
- **Performance**: Caching and lazy evaluation prevent redundant context construction
- **Flexibility**: Same base objects can be evaluated with different contextual augmentations
- **Extensibility**: New context types can be added through the augmentation framework

## Summary

The StackRox policy engine provides a robust, extensible policy evaluation system that:

1. **Centers on workloads** - Every evaluation anchors on a deployment
2. **Enriches with context** - Augments deployments with security-relevant data
3. **Supports complex logic** - Boolean expressions with 85+ criteria fields
4. **Spans the lifecycle** - Build, Deploy, and Runtime evaluation
5. **Enforces comprehensively** - Multiple enforcement mechanisms
6. **Scales efficiently** - Compilation and caching for performance

This architecture enables sophisticated security policy evaluation that considers not just the static configuration of workloads, but their complete operational context including runtime behavior, network relationships, and compliance posture.