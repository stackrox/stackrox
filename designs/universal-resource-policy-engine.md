# Universal Resource Policy Engine for StackRox

## Problem Statement

StackRox's policy engine currently has a fundamental limitation: it can only evaluate policies in the context of deployments. This creates a functional gap compared to other Kubernetes policy engines like Kyverno or ValidatingAdmissionWebhooks, which can enforce policies on any Kubernetes resource.

Customers are forced to use multiple policy engines to achieve their compliance needs, resulting in:
- Operational complexity from managing multiple tools
- Inconsistent policy enforcement and alerting
- Gaps in security coverage
- Poor user experience

Examples of policies that StackRox cannot currently support:
- "No ConfigMaps should contain AWS credentials"
- "All Ingresses must use TLS"
- "Services of type LoadBalancer must have cost-center annotation"
- "PersistentVolumeClaims must not exceed 100Gi"
- "NetworkPolicies must exist for every namespace"

## Current Architecture Limitations

The current StackRox policy engine is deployment-centric:
```
Policy → Deployment → Enhanced with (Images, Network Policies, etc.)
```

Key limitations:
1. All 85 policy criteria fields are predefined and deployment-focused
2. Only workload resources (Deployment, StatefulSet, DaemonSet, etc.) are processed
3. Other resources (ConfigMap, Service, Ingress) are explicitly rejected by admission controller
4. The entire augmentation system assumes a deployment as the root object
5. Alerts and violations are tied to deployment IDs

## Proposed Solution: Universal Resource Policies

### Core Concept

Transform StackRox from a deployment-centric to a resource-centric policy engine:
```
Policy → Any Kubernetes Resource → Enhanced with related context
```

### Design Overview

#### 1. Policy Structure Changes

Extend the policy protobuf to specify target resource types:

```protobuf
message Policy {
  // Existing fields...
  string id = 1;
  string name = 2;
  repeated LifecycleStage lifecycle_stages = 9;
  repeated PolicySection policy_sections = 20;
  
  // NEW: Specify what type of resource this policy evaluates
  ResourceTarget resource_target = 23;
}

message ResourceTarget {
  oneof target {
    DeploymentTarget deployment = 1;  // Legacy, backward compatible
    KubernetesResourceTarget kubernetes = 2;  // NEW
  }
}

message KubernetesResourceTarget {
  string api_version = 1;  // "v1", "networking.k8s.io/v1", etc.
  string kind = 2;         // "ConfigMap", "Service", "Ingress", etc.
  
  // Optional: limit to specific namespaces/names
  repeated string namespaces = 3;
  repeated string names = 4;
}

message DeploymentTarget {
  // Empty for now, allows future deployment-specific options
}
```

#### 2. New Criteria Types

Add dynamic field criteria to evaluate arbitrary Kubernetes fields:

```protobuf
// Option 1: Structured approach
message PolicyGroup {
  string field_name = 1;  // Existing: "CVE", "Privileged Container", etc.
                         // New: "Kubernetes Field", "Related Resource Field"
  
  // When field_name = "Kubernetes Field"
  repeated DynamicFieldValue values = 5;
}

message DynamicFieldValue {
  string field_path = 1;        // "spec.type", "metadata.annotations['key']"
  string operator = 2;          // "equals", "contains", "exists", ">", "regex_match"
  repeated string values = 3;   // Values to match against
}

// Option 2: String-encoded approach (simpler but less structured)
// values = ["field=spec.type,operator=equals,value=LoadBalancer"]
```

Support JSONPath for complex field access:
```
field_path: "$.spec.template.spec.containers[?(@.name=='main')].resources.limits.memory"
```

#### 3. Universal Context Model

Replace deployment-centric augmentation with a universal model:

```go
// Universal augmented object interface
type AugmentedResource interface {
    GetResource() *unstructured.Unstructured
    GetRelatedResources() map[string][]*unstructured.Unstructured
    GetKind() string
    GetAPIVersion() string
    GetNamespace() string
    GetName() string
}

// Specific implementations for common resource types
type AugmentedConfigMap struct {
    ConfigMap *unstructured.Unstructured
    // Workloads using this ConfigMap
    ConsumingWorkloads []*storage.Deployment
    // Other ConfigMaps in same namespace
    RelatedConfigMaps []*unstructured.Unstructured
}

type AugmentedService struct {
    Service *unstructured.Unstructured
    // Endpoints behind this service
    Endpoints *unstructured.Unstructured
    // Workloads selected by this service
    SelectedWorkloads []*storage.Deployment
    // Ingresses routing to this service
    Ingresses []*unstructured.Unstructured
}

type AugmentedIngress struct {
    Ingress *unstructured.Unstructured
    // Backend services
    BackendServices []*unstructured.Unstructured
    // TLS secret if referenced
    TLSSecret *unstructured.Unstructured
    // Workloads behind the ingress (through services)
    BackendWorkloads []*storage.Deployment
}

// Generic fallback for any resource type
type AugmentedGenericResource struct {
    Resource *unstructured.Unstructured
    Related  map[string][]*unstructured.Unstructured
}
```

#### 4. Relationship Discovery

Implement smart relationship discovery based on:
- Owner references
- Label selectors
- Resource specifications (e.g., Service selecting Pods)
- Annotations
- Well-known patterns (e.g., Ingress → Service → Deployment)

```go
type ResourceRelationshipDiscoverer interface {
    DiscoverRelationships(ctx context.Context, 
                         resource *unstructured.Unstructured) (map[string][]*unstructured.Unstructured, error)
}

type RelationshipDiscoverer struct {
    client     kubernetes.Interface
    cache      cache.Store
    strategies map[string]DiscoveryStrategy
}

// Example strategy for ConfigMaps
type ConfigMapDiscoveryStrategy struct{}

func (s *ConfigMapDiscoveryStrategy) Discover(ctx context.Context, 
                                             cm *unstructured.Unstructured) (map[string][]*unstructured.Unstructured, error) {
    related := make(map[string][]*unstructured.Unstructured)
    
    // Find pods using this ConfigMap
    pods := s.findPodsUsingConfigMap(cm)
    related["Pod"] = pods
    
    // Find deployments through pods
    deployments := s.findDeploymentsFromPods(pods)
    related["Deployment"] = deployments
    
    return related, nil
}
```

#### 5. Policy Examples

**ConfigMap Security Policy**:
```yaml
name: "No AWS Credentials in ConfigMaps"
resource_target:
  kubernetes:
    api_version: "v1"
    kind: "ConfigMap"
lifecycle_stages: ["DEPLOY"]
policy_sections:
  - section_name: "Check for AWS credentials"
    policy_groups:
    - field_name: "Kubernetes Field"
      values: 
        - field_path: "data"
          operator: "regex_not_match"
          values: ["AKIA[0-9A-Z]{16}", "aws_secret_access_key"]
```

**Service Policy**:
```yaml
name: "LoadBalancer Services Must Have Cost Center"
resource_target:
  kubernetes:
    api_version: "v1" 
    kind: "Service"
lifecycle_stages: ["DEPLOY"]
policy_sections:
  - section_name: "Check service type and annotation"
    policy_groups:
    - field_name: "Kubernetes Field"
      values:
        - field_path: "spec.type"
          operator: "equals"
          values: ["LoadBalancer"]
    - field_name: "Kubernetes Field"
      values:
        - field_path: "metadata.annotations['billing/cost-center']"
          operator: "exists"
```

**Cross-Resource Policy**:
```yaml
name: "Ingresses Must Point to Services with Network Policies"
resource_target:
  kubernetes:
    api_version: "networking.k8s.io/v1"
    kind: "Ingress"
lifecycle_stages: ["DEPLOY"]
policy_sections:
  - section_name: "Check backend services"
    policy_groups:
    - field_name: "Related Resource Field"
      values:
        - resource_kind: "Service"
          relation: "backend"
          field_path: "hasNetworkPolicy"  # Computed field
          operator: "equals"
          values: ["true"]
```

**Deployment Policy (Backward Compatible)**:
```yaml
name: "High Risk Deployment"
resource_target:
  deployment: {}  # Or omitted for backward compatibility
lifecycle_stages: ["DEPLOY"]
policy_sections:
  - section_name: "Security Context"
    policy_groups:
    - field_name: "Privileged Container"  # Existing criteria work unchanged
      values: ["true"]
```

#### 6. Detection Engine Changes

Create a resource-agnostic detection framework:

```go
// New interface for resource-agnostic detection
type ResourceDetector interface {
    DetectPolicy(ctx context.Context, 
                 resource AugmentedResource, 
                 policy *storage.Policy) (*storage.Alert, error)
}

// Factory to create appropriate detector
func CreateDetector(policy *storage.Policy) ResourceDetector {
    target := policy.GetResourceTarget()
    
    switch t := target.GetTarget().(type) {
    case *storage.ResourceTarget_Deployment:
        return &DeploymentDetector{}  // Existing logic
    case *storage.ResourceTarget_Kubernetes:
        switch t.Kubernetes.GetKind() {
        case "ConfigMap":
            return &ConfigMapDetector{}
        case "Service":
            return &ServiceDetector{}
        default:
            return &GenericResourceDetector{}
        }
    default:
        return &DeploymentDetector{}  // Backward compatibility
    }
}

// Generic detector for any resource
type GenericResourceDetector struct {
    evaluator PolicyEvaluator
}

func (d *GenericResourceDetector) DetectPolicy(ctx context.Context,
                                               resource AugmentedResource,
                                               policy *storage.Policy) (*storage.Alert, error) {
    violations, matched := d.evaluator.Evaluate(resource, policy)
    if !matched {
        return nil, nil
    }
    
    return d.createAlert(resource, policy, violations), nil
}
```

#### 7. Admission Controller Changes

Extend to handle all resource types:

```go
func (m *Manager) ProcessRequest(req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
    // Parse any Kubernetes object (no filtering)
    obj, err := m.unmarshalK8sObject(req)
    if err != nil {
        return nil, err
    }
    
    // Get policies that target this resource type
    policies := m.policySet.GetPoliciesForResource(obj.GetKind(), obj.GetAPIVersion())
    
    // Skip if no policies target this resource
    if len(policies) == 0 {
        return m.allow(), nil
    }
    
    // Augment the resource with context
    augmented, err := m.augmenter.AugmentResource(ctx, obj)
    if err != nil {
        return nil, errors.Wrapf(err, "augmenting %s/%s", obj.GetKind(), obj.GetName())
    }
    
    // Evaluate policies
    var violations []*storage.Alert
    for _, policy := range policies {
        detector := CreateDetector(policy)
        if alert, err := detector.DetectPolicy(ctx, augmented, policy); err != nil {
            log.Errorf("Error detecting policy %s: %v", policy.GetName(), err)
        } else if alert != nil {
            violations = append(violations, alert)
        }
    }
    
    return m.buildResponse(violations)
}
```

#### 8. Storage and Alert Changes

Extend alert model to handle non-deployment resources:

```protobuf
message Alert {
  // Existing fields...
  string id = 1;
  Policy policy = 4;
  
  // Change from deployment_id to resource reference
  oneof entity {
    string deployment_id = 13 [deprecated = true];
    ResourceReference resource = 50;  // NEW
  }
}

message ResourceReference {
  string api_version = 1;
  string kind = 2;
  string namespace = 3;
  string name = 4;
  string uid = 5;
}
```

## Implementation Phases

### Phase 1: Foundation (2-3 months)
1. Extend policy protobuf with ResourceTarget
2. Create universal AugmentedResource interface
3. Implement generic resource augmentation
4. Add "Kubernetes Field" criteria type
5. Create GenericResourceDetector

### Phase 2: Core Resources (2-3 months)
1. Implement specialized augmenters for common resources:
   - ConfigMap, Secret, Service, Ingress, NetworkPolicy
2. Add relationship discovery strategies
3. Extend admission controller to process all resources
4. Update storage layer for non-deployment alerts

### Phase 3: Advanced Features (2-3 months)
1. Add JSONPath support for complex field queries
2. Implement cross-resource policy criteria
3. Support for Custom Resources (CRDs)
4. Performance optimizations for resource caching

### Phase 4: UI and UX (1-2 months)
1. Update UI to display non-deployment violations
2. Create resource-specific violation views
3. Add policy creation UI for new criteria types

## Benefits

1. **Complete Kubernetes Coverage**: Enforce policies on any Kubernetes resource
2. **Single Policy Engine**: Eliminate need for multiple tools
3. **Backward Compatibility**: Existing deployment policies continue working
4. **Extensibility**: Easy to add new resource types and relationships
5. **Market Differentiation**: First security platform with universal K8s policies

## Risks and Mitigations

### Risk 1: Performance Impact
- **Risk**: Evaluating all resources could overwhelm the system
- **Mitigation**: Implement resource filtering, caching, and lazy evaluation

### Risk 2: Storage Growth
- **Risk**: Storing alerts for all resources could explode storage
- **Mitigation**: Implement retention policies, aggregation, and pruning

### Risk 3: Complexity
- **Risk**: System becomes too complex to maintain
- **Mitigation**: Modular design, extensive testing, gradual rollout

### Risk 4: Breaking Changes
- **Risk**: Existing integrations break
- **Mitigation**: Careful API versioning, deprecation notices, compatibility layer

## Alternative Approaches Considered

1. **Webhook-Only Approach**: Add a generic webhook for other tools
   - Rejected: Doesn't provide unified experience

2. **OPA Integration**: Embed OPA for generic policies
   - Rejected: Adds complexity, different policy language

3. **Limited Extension**: Only add a few more resource types
   - Rejected: Doesn't solve the fundamental limitation

## Conclusion

This proposal transforms StackRox from a container security platform to a comprehensive Kubernetes policy engine. By making the policy engine resource-agnostic, we can provide customers with a single, unified solution for all their Kubernetes policy needs, eliminating the operational complexity of managing multiple policy engines.

The design maintains backward compatibility while opening up entirely new use cases, positioning StackRox as the most comprehensive security and compliance solution for Kubernetes.