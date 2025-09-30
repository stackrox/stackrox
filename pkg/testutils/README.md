# StackRox E2E Testing Framework - Phase 1

This package provides a comprehensive Ginkgo/Gomega-based testing framework for StackRox E2E tests, implementing Phase 1 of the Groovy to Go migration strategy.

## Overview

The framework provides:
- **Centralized credential management** with CI/CD-aware modes
- **Comprehensive StackRox client libraries** for all API services
- **External service integration** with automatic mock/real service selection
- **Simple chaos engineering** for admission controller resilience testing
- **Complete Ginkgo BDD integration** with resource management using DeferCleanup

## Quick Start

```go
package mytest

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    ginkgoHelper "github.com/stackrox/rox/pkg/testutils/ginkgo"
)

var _ = Describe("My Test Suite", func() {
    var suite *ginkgoHelper.StackRoxTestSuite

    BeforeEach(func() {
        suite = ginkgoHelper.NewStackRoxTestSuite(nil) // Uses default config
        suite.SetupSuite()
    })

    It("should create and test a policy", func() {
        policyConfig := &clients.PolicyConfig{
            Name: "Test Policy",
            Categories: []string{"Container Security"},
            Enforcement: true,
            Scope: clients.RuntimeScope,
        }

        policy := suite.CreateTestPolicy(context.Background(), policyConfig)
        // Policy is automatically cleaned up via DeferCleanup

        // Test policy enforcement...
    })
})
```

## Components

### 1. Credentials Management (`pkg/testutils/credentials/`)

Centralizes all external service credentials with CI/CD-aware validation:

```go
creds, err := credentials.Load()
if err != nil {
    // Handle error
}

// Check credential availability
if creds.HasGCRCredentials() {
    // Use real GCR client
} else if creds.IsDevelopmentMode() {
    // Use mock GCR client
}
```

**Supported Test Environments:**
- `development`: Local development with mock external services
- `ci-pr`: PR builds with mocked external services for speed/reliability
- `ci-master`: Master/nightly builds with real external services for full E2E validation

### 2. StackRox Client Libraries (`pkg/testutils/clients/`)

Comprehensive API client wrappers with test-friendly interfaces:

```go
clients := clients.NewStackRoxClients(t, creds)
defer clients.Close()

// Policy management
policy, err := clients.Policies.CreatePolicy(ctx, &PolicyConfig{...})

// Alert monitoring with wait capabilities
alerts, err := clients.Alerts.WaitForPolicyAlerts(ctx, policyID, minCount)

// Image scanning
scanResult, err := clients.Images.ScanImage(ctx, "nginx:latest")
```

**Available Clients:**
- `Policies`: Policy creation, management, and enforcement configuration
- `Alerts`: Alert monitoring with waiting mechanisms and filtering
- `Images`: Image scanning and vulnerability management
- `Clusters`: Cluster status and management
- `Auth`: Authentication and authorization
- `Backups`: Backup operations and storage integration

### 3. External Service Integration (`pkg/testutils/external/`)

Automatic mock/real service selection based on credential availability:

```go
// Container registries
registryClient, err := external.NewRegistryClient(creds, external.GCRRegistry)
scanResult, err := registryClient.ScanImage("gcr.io/test/app:latest")

// Notifications
notifClient, err := external.NewNotificationClient(creds, external.SlackNotification)
err = notifClient.SendMessage(&NotificationMessage{...})

// Cloud storage
storageClient, err := external.NewStorageClient(creds, external.S3Storage)
result, err := storageClient.UploadBackup("backup-name", dataReader)
```

**Supported Services:**
- **Container Registries**: GCR, ECR, ACR, Quay.io, Red Hat Registry
- **Notifications**: Slack, Email, Generic Webhooks, Splunk
- **Cloud Storage**: AWS S3, Google Cloud Storage, Azure Blob Storage

### 4. Chaos Engineering (`pkg/testutils/chaos/`)

Simple chaos monkey for admission controller resilience testing:

```go
chaosConfig := chaos.DefaultAdmissionControllerConfig()
chaosMonkey := chaos.NewAdmissionControllerChaos(k8sClient, chaosConfig)

err := chaosMonkey.Start(ctx)
// Chaos monkey kills admission controller pods at intervals

// Wait for recovery
err = chaosMonkey.WaitForPodRecovery(ctx, 2*time.Minute)
```

### 5. Ginkgo BDD Integration (`pkg/testutils/ginkgo/`)

Complete test suite with resource management and custom matchers:

```go
suite := ginkgoHelper.NewStackRoxTestSuite(config)

// BDD helpers
suite.GivenPolicy("Test Policy", true, "Container Security")
suite.WhenDeploymentCreated("test-deployment", "nginx:latest")
suite.ThenAlertGenerated(policyID, 1)

// Custom matchers
Expect(alerts).To(HaveAlerts(2).WithSeverity(storage.Severity_HIGH_SEVERITY))
Expect(scanResult).To(HaveVulnerability().WithCVE("CVE-2023-1234"))
Expect(deploymentName).To(BeDeploymentBlocked())
```

## Policy Field Testing Pattern (173 Scenarios)

The framework supports migrating the complex PolicyFieldsTest.groovy with its 173 scenarios:

```go
DescribeTable("Runtime enforcement scenarios",
    func(category string, enforcement bool, expectedBehavior PolicyBehavior) {
        // Test implementation
    },

    // 173 entries covering all policy combinations
    Entry("Privilege Escalation + Enforce", "Privilege Escalation", true, PolicyBehavior{
        ShouldBlock: true, ShouldAlert: true, ExpectedSeverity: "HIGH",
    }),
    Entry("Container Security + Monitor", "Container Security", false, PolicyBehavior{
        ShouldBlock: false, ShouldAlert: true, ExpectedSeverity: "MEDIUM",
    }),
    // ... 171 more entries
)
```

## CI/CD Integration

### Local Development
```bash
export ROX_TEST_ENV=development
export ROX_ADMIN_PASSWORD=your-password
# All external services use mocks - fast iteration (~45-60 minutes)
ginkgo --label-filter="fast" ./tests/...
```

### PR Builds
```bash
export ROX_TEST_ENV=ci-pr
export ROX_ADMIN_PASSWORD=ci-password
# StackRox APIs real, external services mocked - reliability (~75-90 minutes)
ginkgo ./tests/...
```

### Master/Nightly Builds
```bash
export ROX_TEST_ENV=ci-master
export ROX_ADMIN_PASSWORD=ci-password
export AWS_ACCESS_KEY_ID=ci-key
export GOOGLE_CREDENTIALS_GCR_SCANNER_V2=ci-creds
# All services real - full E2E validation (~105 minutes)
ginkgo ./tests/...
```

## Performance Expectations

- **Local Development**: ~45-60 minutes (subset execution with mocks)
- **PR Builds**: ~75-90 minutes (modest improvement through external service reliability)
- **Master/Nightly**: ~105 minutes (maintain Groovy baseline with real services)

Primary performance improvements come from:
- Elimination of external service flakiness in PR builds
- Better resource cleanup preventing resource leaks
- Parallel execution where admission controller allows
- More reliable timing assertions using Eventually/Consistently

## Resource Management

All components use Ginkgo's DeferCleanup for guaranteed resource cleanup:

```go
// Automatic policy cleanup
policy := suite.CreateTestPolicy(ctx, config)
// DeferCleanup automatically registered

// Manual cleanup registration
DeferCleanup(func() {
    teardownDeployment("test-deployment")
})
```

## Migration Strategy

This Phase 1 framework enables the three-phase migration approach:

1. **Phase 1 (Complete)**: Framework implementation with all infrastructure
2. **Phase 2 (Next)**: Manual migration of 1-2 representative tests to establish patterns
3. **Phase 3 (Future)**: Sub-agent bulk migration using established patterns and framework

The framework is designed to support both manual migration and AI-assisted conversion with human review.

## Example Usage

See `tests/e2e/ginkgo_example_test.go` for a comprehensive example demonstrating:
- PolicyFieldsTest migration patterns
- External service integration
- Chaos engineering
- Custom matchers and BDD helpers
- Proper resource cleanup

This example provides the foundation for migrating the remaining ~53 active Groovy test files using established patterns and comprehensive framework infrastructure.