# PolicyReport Integration Design for StackRox

## Executive Summary

This design proposes integrating StackRox with the Kubernetes PolicyReport API to enable interoperability with other policy engines like Kyverno, Falco, and OPA. The integration will allow StackRox to both produce and consume PolicyReports, providing a unified compliance view across multiple policy engines.

## Background

### Current State
- StackRox operates as an isolated policy engine with its own alert format
- Customers using multiple policy engines (Kyverno, OPA, etc.) must manage separate dashboards and alert streams
- No standardized way to share policy violations between different tools

### PolicyReport API
The PolicyReport API is a Kubernetes Working Group standard (wgpolicyk8s.io) for representing policy evaluation results. It defines:
- **PolicyReport**: Namespaced resource for policy results
- **ClusterPolicyReport**: Cluster-scoped policy results
- **Standardized format**: Source, policy, rule, severity, resources, etc.

## Design Overview

### Architecture

Implement a **PolicyReport Controller** in the Secured Cluster that provides bidirectional synchronization:

```
StackRox Alerts → PolicyReport Producer → PolicyReport CRs
                                              ↓
Other Tools → PolicyReport CRs → PolicyReport Consumer → Central
                                                           ↓
                                                    Unified Dashboard
```

### Key Components

1. **PolicyReport Producer**: Converts StackRox alerts to PolicyReport CRs
2. **PolicyReport Consumer**: Watches PolicyReports from other tools and sends to Central
3. **Central API Extension**: New endpoints to accept external policy alerts
4. **Extended Alert Model**: Support for external policy metadata

## Detailed Design

### 1. Alert Model Extension

Extend the Alert protobuf to support external policies:

```protobuf
message Alert {
  // Existing fields...
  string id = 1;
  
  oneof policy_source {
    Policy stackrox_policy = 4;           // Existing StackRox policy
    ExternalPolicy external_policy = 51;  // NEW: External policy metadata
  }
  
  oneof entity {
    string deployment_id = 13 [deprecated = true];
    ResourceReference resource = 50;  // From universal resource design
  }
}

message ExternalPolicy {
  string source = 1;        // "kyverno", "falco", "opa"
  string name = 2;          // Policy name
  string rule = 3;          // Specific rule that failed
  string category = 4;      // Policy category
  
  // Store the original policy definition if available
  string policy_yaml = 5;   // Original policy YAML for context
  
  // Structured criteria representation for rich UX
  repeated PolicyCriteria criteria = 6;
}

message PolicyCriteria {
  string field = 1;         // What was checked
  string operator = 2;      // How it was checked  
  string expected = 3;      // Expected value
  string actual = 4;        // Actual value
  string description = 5;   // Human-readable explanation
}
```

### 2. PolicyReport Controller Implementation

Located in Secured Cluster for local CRD management:

```go
package policyreport

type Controller struct {
    k8sClient     kubernetes.Interface
    policyClient  wgpolicy.Interface
    centralClient central.AlertServiceClient
    clusterID     string
    
    producer      *Producer
    consumer      *Consumer
}

// Watches StackRox alerts and creates PolicyReports
type Producer struct {
    alertStream   central.AlertService_GetAlertsClient
    policyClient  wgpolicy.Interface
    
    // Cache to prevent duplicate reports
    reportCache   cache.Store
}

// Watches PolicyReports and sends to Central
type Consumer struct {
    informer      cache.SharedIndexInformer
    centralClient central.AlertServiceClient
    clusterID     string
    
    // Track which reports we've already sent
    processedReports sets.String
}
```

### 3. PolicyReport Production

Convert StackRox alerts to PolicyReports:

```go
func (p *Producer) alertToPolicyReport(alert *storage.Alert) *wgpolicy.PolicyReport {
    return &wgpolicy.PolicyReport{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("stackrox-%s", alert.GetId()),
            Namespace: alert.GetResource().GetNamespace(),
            Labels: map[string]string{
                "app.kubernetes.io/managed-by": "stackrox",
                "policy.stackrox.io/severity":  alert.GetPolicy().GetSeverity().String(),
            },
        },
        Spec: wgpolicy.PolicyReportSpec{
            Scope: &corev1.ObjectReference{
                APIVersion: alert.GetResource().GetApiVersion(),
                Kind:       alert.GetResource().GetKind(),
                Name:       alert.GetResource().GetName(),
                Namespace:  alert.GetResource().GetNamespace(),
            },
            Results: []wgpolicy.PolicyReportResult{
                {
                    Source:   "stackrox",
                    Policy:   alert.GetPolicy().GetName(),
                    Rule:     alert.GetPolicy().GetPolicySections()[0].GetSectionName(),
                    Result:   wgpolicy.PolicyResult("fail"),
                    Severity: mapSeverity(alert.GetPolicy().GetSeverity()),
                    Category: alert.GetPolicy().GetCategories()[0],
                    Message:  alert.GetViolationMessage(),
                    Resources: []corev1.ObjectReference{
                        {
                            APIVersion: alert.GetResource().GetApiVersion(),
                            Kind:       alert.GetResource().GetKind(),
                            Name:       alert.GetResource().GetName(),
                            Namespace:  alert.GetResource().GetNamespace(),
                        },
                    },
                    Properties: map[string]string{
                        "alertId":           alert.GetId(),
                        "lifecycleStage":    alert.GetLifecycleStage().String(),
                        "enforcementAction": alert.GetEnforcement().String(),
                    },
                    Timestamp: metav1.NewTime(alert.GetTime().AsTime()),
                },
            },
        },
    }
}
```

### 4. PolicyReport Consumption

Process PolicyReports from other tools:

```go
func (c *Consumer) processPolicyReport(report *wgpolicy.PolicyReport) error {
    // Skip reports created by StackRox
    if report.Labels["app.kubernetes.io/managed-by"] == "stackrox" {
        return nil
    }
    
    for _, result := range report.Spec.Results {
        if result.Result != "pass" {
            alert := c.policyReportToAlert(report, result)
            
            req := &central.CreateExternalAlertRequest{
                Source:    result.Source,
                ClusterId: c.clusterID,
                Alert:     alert,
            }
            
            if _, err := c.centralClient.CreateExternalAlert(ctx, req); err != nil {
                return errors.Wrapf(err, "sending alert to central")
            }
        }
    }
    
    return nil
}

func (c *Consumer) policyReportToAlert(report *wgpolicy.PolicyReport, 
                                       result wgpolicy.PolicyReportResult) *storage.Alert {
    return &storage.Alert{
        PolicySource: &storage.Alert_ExternalPolicy{
            ExternalPolicy: &storage.ExternalPolicy{
                Source:     result.Source,
                Name:       result.Policy,
                Rule:       result.Rule,
                Category:   result.Category,
                PolicyYaml: result.Properties["policyDefinition"],
                Criteria:   extractCriteria(result),
            },
        },
        Resource: &storage.ResourceReference{
            ApiVersion: report.Spec.Scope.APIVersion,
            Kind:       report.Spec.Scope.Kind,
            Name:       report.Spec.Scope.Name,
            Namespace:  report.Spec.Scope.Namespace,
        },
        Severity: mapPolicyReportSeverity(result.Severity),
        Message:  result.Message,
        Time:     protocompat.ConvertTimeToTimestamp(result.Timestamp.Time),
    }
}
```

### 5. Central API Extension

New service methods for external alerts:

```protobuf
service AlertService {
  // Existing methods...
  
  // Create alert from external policy engine
  rpc CreateExternalAlert(CreateExternalAlertRequest) returns (Alert) {
    option (google.api.http) = {
      post: "/v1/externalAlerts"
      body: "*"
    };
  }
  
  // List external policy sources
  rpc ListExternalPolicySources(Empty) returns (ListExternalPolicySourcesResponse) {
    option (google.api.http) = {
      get: "/v1/externalAlerts/sources"
    };
  }
}

message CreateExternalAlertRequest {
  string source = 1;     // Policy engine source
  string cluster_id = 2; // Source cluster
  Alert alert = 3;       // The alert details
}

message ListExternalPolicySourcesResponse {
  repeated string sources = 1; // ["kyverno", "falco", "opa"]
}
```

### 6. UI/UX Enhancements

#### Alert List View
- Show source badge (StackRox/Kyverno/Falco icon)
- Unified severity and category display
- Filter by policy source

#### Alert Details View
- **For StackRox Alerts**: Existing detailed policy view
- **For External Alerts**: 
  - Display stored policy YAML with syntax highlighting
  - Show criteria breakdown if available
  - Link to source tool if URL provided
  - "Why did this fail?" explanation from criteria

#### Example UI Mock:
```
┌─────────────────────────────────────────────────┐
│ Alert: Deployment nginx violates policy         │
├─────────────────────────────────────────────────┤
│ Source: [Kyverno icon] Kyverno                 │
│ Policy: require-pod-security-standards          │
│ Rule: check-security-context                    │
│ Severity: High                                  │
│                                                 │
│ Policy Definition:                              │
│ ┌─────────────────────────────────────────────┐ │
│ │ spec:                                       │ │
│ │   validationFailureAction: enforce          │ │
│ │   rules:                                    │ │
│ │   - name: check-security-context            │ │
│ │     match:                                  │ │
│ │       any:                                  │ │
│ │       - resources:                          │ │
│ │           kinds:                            │ │
│ │           - Pod                             │ │
│ │     validate:                               │ │
│ │       message: "Security context required"  │ │
│ │       pattern:                              │ │
│ │         spec:                               │ │
│ │           securityContext:                  │ │
│ │             runAsNonRoot: true              │ │
│ └─────────────────────────────────────────────┘ │
│                                                 │
│ What Failed:                                    │
│ ✗ Field: spec.securityContext.runAsNonRoot     │
│   Expected: true                                │
│   Actual: <not set>                             │
└─────────────────────────────────────────────────┘
```

## Implementation Plan

### Phase 1: Foundation (4-6 weeks)
Prerequisites: Universal Resource Policy Engine (for non-deployment alerts)

- [ ] Extend Alert protobuf with ExternalPolicy
- [ ] Implement PolicyReport Producer in Secured Cluster
- [ ] Create Central API for external alerts
- [ ] Basic storage and retrieval

### Phase 2: Consumption (4-6 weeks)
- [ ] Implement PolicyReport Consumer
- [ ] Add criteria extraction logic
- [ ] Policy YAML storage
- [ ] Central-side alert processing

### Phase 3: UI Integration (3-4 weeks)
- [ ] Update alert list for multi-source
- [ ] External policy detail views
- [ ] Source filtering and grouping
- [ ] Policy criteria visualization

### Phase 4: Advanced Features (2-3 weeks)
- [ ] Historical report tracking
- [ ] Cross-engine correlation
- [ ] Compliance reporting integration
- [ ] Performance optimization

## Benefits

1. **Unified Compliance**: Single pane of glass for all policy violations
2. **Ecosystem Integration**: Works with any PolicyReport-compliant tool
3. **Rich Context**: Store and display external policy details
4. **Standards Compliance**: Adopts Kubernetes community standards
5. **Reduced Complexity**: Eliminates need for multiple dashboards

## Risks and Mitigations

### Risk 1: Performance Impact
- **Risk**: High volume of PolicyReports could overwhelm system
- **Mitigation**: Rate limiting, batching, and selective watching

### Risk 2: Storage Growth
- **Risk**: Storing policy YAML could consume significant space
- **Mitigation**: Compression, retention policies, optional storage

### Risk 3: Schema Evolution
- **Risk**: PolicyReport API might change
- **Mitigation**: Version detection, graceful degradation

## Alternative Approaches Considered

### 1. Aggregation API Server
- **Rejected**: Unnecessary complexity for simple CRUD operations
- CRD watching is simpler and follows Kubernetes patterns

### 2. Direct Integration with Each Tool
- **Rejected**: Would require custom code for each policy engine
- PolicyReport provides a standard interface

### 3. One-Way Integration Only
- **Rejected**: Bidirectional provides more value
- Users want both unified viewing and standard reporting

## Conclusion

This PolicyReport integration positions StackRox as a central hub for Kubernetes policy compliance, working seamlessly with the broader ecosystem while maintaining its strengths in runtime security and vulnerability management. The bidirectional approach provides maximum value with reasonable implementation complexity.