# StackRox E2E Testing Framework - Phase 1

This package provides a comprehensive Ginkgo/Gomega-based testing framework for StackRox E2E tests, implementing Phase 1 of the Groovy to Go migration strategy.

## Overview

The framework provides:

* **Centralized environment variable management** using the `env.Setting` pattern
* **External service integration** with automatic mock/real service selection
* **Generic chaos engineering** for Kubernetes pod resilience testing
* **Custom Gomega matchers** for StackRox-specific assertions
* **Complete Ginkgo BDD integration** with resource management using DeferCleanup

## Quick Start

```go
//go:build test_e2e

package e2e

import (
    "context"
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    v1 "github.com/stackrox/rox/generated/api/v1"
    "github.com/stackrox/rox/pkg/testutils/centralgrpc"
    ginkgoHelper "github.com/stackrox/rox/pkg/testutils/ginkgo"
    "google.golang.org/grpc"
)

func TestE2E(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "E2E Suite")
}

var _ = Describe("My Test Suite", func() {
    var (
        conn      *grpc.ClientConn
        policySvc v1.PolicyServiceClient
        alertSvc  v1.AlertServiceClient
    )

    BeforeEach(func() {
        conn = centralgrpc.GRPCConnectionToCentral(GinkgoT())
        policySvc = v1.NewPolicyServiceClient(conn)
        alertSvc = v1.NewAlertServiceClient(conn)

        DeferCleanup(func() {
            conn.Close()
        })
    })

    It("should create and test a policy", func() {
        policy, err := policySvc.PostPolicy(ctx, &v1.PostPolicyRequest{...})
        Expect(err).NotTo(HaveOccurred())

        DeferCleanup(func() {
            policySvc.DeletePolicy(context.Background(), &v1.ResourceByID{Id: policy.GetId()})
        })

        // Test policy enforcement...
    })
})
```

## Components

### 1. Environment Variable Management (`pkg/testutils/env/`)

Centralized test environment variable management using the same `env.Setting` pattern as production code:

```go
import testenv "github.com/stackrox/rox/pkg/testutils/env"

// Settings are registered at package level
username := testenv.ROXUsername.Setting()
password := testenv.ROXAdminPassword.Setting()

// Check credential availability
if testenv.HasAWSCredentials() {
    // Use real AWS services
} else if testenv.ShouldUseMockServices() {
    // Use mock services
}
```

**Key Settings:**

* `ROXUsername`, `ROXAdminPassword` - StackRox authentication
* `APIHostname`, `APIEndpoint` - StackRox API connection
* `AWSAccessKeyID`, `AWSSecretAccessKey` - AWS credentials
* `GCPServiceAccount`, `GoogleGCRCredentials` - GCP credentials
* `AzureClientID`, `AzureClientSecret`, `AzureTenantID` - Azure credentials
* `SlackWebhookURL`, `EmailSMTPServer` - Notification services
* `RegistryUsername`, `RegistryPassword` - Container registry credentials
* `ShouldUseMockServices` - Force mock mode for all external services

**Design:**

* Reuses production `env.Setting` infrastructure from `pkg/env`
* Only `ROX_ADMIN_PASSWORD` is shared between production and test code
* All other settings are test-specific and registered in `pkg/testutils/env`
* External packages import as `testenv` but internal code uses `env`

### 2. External Service Integration (`pkg/testutils/external/`)

Automatic mock/real service selection based on credential availability. Each client constructor handles its own credentials and embeds mock logic:

**Container Registries** (`registry.go`):

```go
// Each registry client pulls its own credentials
gcrClient, err := external.NewGCRClient()
scanResult, err := gcrClient.ScanImage("gcr.io/test/app:latest")

ecrClient, err := external.NewECRClient()
acrClient, err := external.NewACRClient()
quayClient, err := external.NewQuayClient()
redhatClient, err := external.NewRedHatClient()
```

**Notification Services** (`notification.go`):

```go
slackClient, err := external.NewSlackClient()
err = slackClient.SendMessage(&external.NotificationMessage{
    Title:    "Alert",
    Text:     "Policy violation detected",
    Channel:  "#security",
    Severity: "HIGH",
})

emailClient, err := external.NewEmailClient()
webhookClient, err := external.NewWebhookClient()
splunkClient, err := external.NewSplunkClient()
```

**Cloud Storage** (`storage.go`):

```go
s3Client, err := external.NewS3Client()
result, err := s3Client.UploadBackup("backup-name", dataReader)
backups, err := s3Client.ListBackups()
err = s3Client.DeleteBackup("backup-name")

gcsClient, err := external.NewGCSClient()
azureClient, err := external.NewAzureClient()
```

**Design:**

* No factory functions with case statements
* Each `New*Client()` constructor pulls credentials directly from `testenv`
* Mock logic embedded in each client type (mock bool field + mockClient)
* Constructors return concrete types, not interfaces (following Go idioms)
* No unused interfaces (removed per YAGNI principle)

### 3. Chaos Engineering (`pkg/testutils/chaos/`)

Generic pod chaos testing for Kubernetes resilience, decoupled from specific components:

```go
// Create custom chaos configuration
chaosConfig := &chaos.Config{
    Namespace:    "stackrox",
    PodSelector:  map[string]string{"app": "my-service"},
    KillInterval: 3 * time.Minute,
    MaxKills:     5,
    Enabled:      true,
}
podChaos := chaos.NewPodChaos(k8sClient, chaosConfig)

// Or use convenience builders for StackRox components
admissionChaos := chaos.NewPodChaos(k8sClient, chaos.AdmissionControllerConfig("stackrox"))
sensorChaos := chaos.NewPodChaos(k8sClient, chaos.SensorConfig("stackrox"))
centralChaos := chaos.NewPodChaos(k8sClient, chaos.CentralConfig("stackrox"))

// Start chaos monkey
err := podChaos.Start(ctx)

// Run test wrapper with chaos
wrapper := chaos.NewTestWrapper(k8sClient, chaosConfig, 5*time.Minute)
err = wrapper.RunWithChaos(ctx, func(ctx context.Context) error {
    // Your test code here - chaos monkey is active
    return nil
})
```

**Design:**

* Generic `PodChaos` type works with any Kubernetes pods
* Chaos functionality decoupled from specific components
* Convenience builders (`AdmissionControllerConfig()`, `SensorConfig()`, `CentralConfig()`) provide sensible defaults
* Renamed from `AdmissionControllerChaos` to `PodChaos` for clarity

### 4. Ginkgo BDD Integration (`pkg/testutils/ginkgo/`)

Custom Gomega matchers for StackRox-specific assertions:

```go
import ginkgoHelper "github.com/stackrox/rox/pkg/testutils/ginkgo"

// Alert matchers
Expect(alerts).To(ginkgoHelper.HaveAlerts(2))
Expect(alerts).To(ginkgoHelper.HaveAlert().WithSeverity(storage.Severity_HIGH_SEVERITY))

// Vulnerability matchers
Expect(scanResult).To(ginkgoHelper.HaveVulnerability().WithCVE("CVE-2023-1234"))
Expect(scanResult).To(ginkgoHelper.HaveVulnerability().WithSeverity("CRITICAL").WithMinCVSS(9.0))

// Deployment blocking matcher
Expect(deploymentName).To(ginkgoHelper.BeDeploymentBlocked())

// Notification matcher
err := notifClient.SendMessage(msg)
Expect(err).To(ginkgoHelper.BeSuccessfulNotification())

// Eventually helpers
ginkgoHelper.EventuallyHaveAlert(func() []*storage.Alert {
    resp, _ := alertSvc.ListAlerts(ctx, &v1.ListAlertsRequest{...})
    return resp.Alerts
}, 1, 2*time.Minute)

ginkgoHelper.ConsistentlyNoAlert(func() []*storage.Alert {
    resp, _ := alertSvc.ListAlerts(ctx, &v1.ListAlertsRequest{...})
    return resp.Alerts
}, 30*time.Second)

// BDD helper functions for test descriptions
given := ginkgoHelper.GivenPolicyWithEnforcement("Test Policy", true)
when := ginkgoHelper.WhenDeploymentViolatesPolicy("test-app", "Container Security")
then := ginkgoHelper.ThenExpectBehavior(true, true, "HIGH")
// Outputs: "Then should block deployment and generate HIGH alert"
```

**Available Matchers:**

* `HaveAlert()`, `HaveAlerts(count)` - Check alert count and severity
* `HaveVulnerability()` - Check scan results for vulnerabilities
* `BeDeploymentBlocked()` - Check if deployment is blocked by policy
* `BeSuccessfulNotification()` - Check notification delivery success

**Design:**

* Custom matchers follow Gomega naming conventions (`Be*`, `Have*`)
* Helper functions execute assertions directly (no return values)
* BDD helper functions create descriptive strings for test documentation

### 5. gRPC Connection Management (`pkg/testutils/centralgrpc/`)

Simplified connection pattern using raw gRPC service clients:

```go
BeforeEach(func() {
    // Single connection per test suite
    conn = centralgrpc.GRPCConnectionToCentral(GinkgoT())

    // Multiple service clients share the connection via HTTP/2 multiplexing
    policySvc = v1.NewPolicyServiceClient(conn)
    alertSvc = v1.NewAlertServiceClient(conn)
    imageSvc = v1.NewImageServiceClient(conn)

    DeferCleanup(func() {
        conn.Close()
    })
})
```

**Design:**

* Use raw gRPC service clients (`v1.New*ServiceClient()`) instead of custom wrappers
* One connection per test suite shared across all service clients
* HTTP/2 multiplexing handles concurrent requests efficiently
* No custom client wrappers (removed `pkg/testutils/clients/` - YAGNI)

## Policy Field Testing Pattern (173 Scenarios)

The framework supports migrating the complex PolicyFieldsTest.groovy with its 173 scenarios using Ginkgo's table-driven testing:

```go
Context("Runtime Policy Categories", func() {
    DescribeTable("Runtime enforcement scenarios",
        func(category string, enforcement bool, expectedBehavior PolicyBehavior) {
            By(fmt.Sprintf("Given a %s policy with enforcement=%v", category, enforcement))

            policy, err := policySvc.PostPolicy(ctx, &v1.PostPolicyRequest{...})
            Expect(err).NotTo(HaveOccurred())
            DeferCleanup(func() {
                policySvc.DeletePolicy(context.Background(), &v1.ResourceByID{Id: policy.GetId()})
            })

            By("When deploying a violating application")
            // Create deployment...

            By(fmt.Sprintf("Then should %s", expectedBehavior.Description))

            if expectedBehavior.ShouldAlert {
                Eventually(func() []*storage.ListAlert {
                    resp, _ := alertSvc.ListAlerts(ctx, &v1.ListAlertsRequest{...})
                    return resp.Alerts
                }, 2*time.Minute, 10*time.Second).Should(HaveLen(1))
            }

            if expectedBehavior.ShouldBlock {
                Eventually(func() string {
                    return deploymentName
                }, 2*time.Minute, 10*time.Second).Should(
                    ginkgoHelper.BeDeploymentBlocked(),
                )
            }
        },

        // 173 entries covering all policy combinations
        Entry("Privilege Escalation + Enforce", "Privilege Escalation", true, PolicyBehavior{
            ShouldBlock:      true,
            ShouldAlert:      true,
            ExpectedSeverity: storage.Severity_HIGH_SEVERITY,
            Description:      "block deployment and generate alert",
            AlertTimeout:     2 * time.Minute,
            BlockingTimeout:  2 * time.Minute,
        }),

        Entry("Privilege Escalation + Monitor", "Privilege Escalation", false, PolicyBehavior{
            ShouldBlock:      false,
            ShouldAlert:      true,
            ExpectedSeverity: storage.Severity_HIGH_SEVERITY,
            Description:      "allow deployment but generate alert",
            AlertTimeout:     2 * time.Minute,
        }),

        // ... 171 more entries
    )
})
```

## Running Tests

### Prerequisites

```bash
# Required for all test runs
export ROX_ADMIN_PASSWORD=your-password

# Optional: Force mock services (default if no credentials)
export SHOULD_USE_MOCK_SERVICES=true
```

### Local Development

```bash
# Run all E2E tests
go test -tags test_e2e ./tests/e2e

# Run with verbose output
go test -v -tags test_e2e ./tests/e2e

# Run specific test
go test -tags test_e2e ./tests/e2e -ginkgo.focus="Policy Field Validation"

# Limit parallelism (for resource-constrained environments)
go test -p 2 -tags test_e2e ./tests/e2e
```

### Using Ginkgo CLI

```bash
# Install Ginkgo CLI
go install github.com/onsi/ginkgo/v2/ginkgo

# Run tests with color output and progress
ginkgo -v ./tests/e2e

# Run in parallel
ginkgo -p ./tests/e2e

# Run with labels
ginkgo --label-filter="fast" ./tests/e2e
```

## CI/CD Integration

### Local Development Mode

```bash
export SHOULD_USE_MOCK_SERVICES=true
export ROX_ADMIN_PASSWORD=your-password
# All external services use mocks - fast iteration
go test -tags test_e2e ./tests/e2e
```

### PR Builds

```bash
export SHOULD_USE_MOCK_SERVICES=true
export ROX_ADMIN_PASSWORD=ci-password
# Mock external services for reliability
go test -tags test_e2e ./tests/...
```

### Master/Nightly Builds

```bash
export ROX_ADMIN_PASSWORD=ci-password
export AWS_ACCESS_KEY_ID=ci-key
export AWS_SECRET_ACCESS_KEY=ci-secret
export GOOGLE_CREDENTIALS_GCR_SCANNER_V2=ci-creds
export SLACK_WEBHOOK_URL=https://hooks.slack.com/...
# All services real - full E2E validation
go test -tags test_e2e ./tests/...
```

## Resource Management

All components use Ginkgo's DeferCleanup for guaranteed resource cleanup:

```go
// Automatic cleanup in tests
policy, err := policySvc.PostPolicy(ctx, &v1.PostPolicyRequest{...})
Expect(err).NotTo(HaveOccurred())

DeferCleanup(func() {
    policySvc.DeletePolicy(context.Background(), &v1.ResourceByID{Id: policy.GetId()})
})

// Cleanup runs even if test fails
// Cleanup runs in LIFO order (reverse of registration)
```

## Architecture Decisions

### env.Setting Pattern

* **Reuses production infrastructure**: Same pattern as `pkg/env/setting.go`
* **Single source of truth**: `ROX_ADMIN_PASSWORD` shared via `env.PasswordEnv`
* **Test-specific settings**: All other credentials registered in `pkg/testutils/env`
* **No custom structs**: Package-level functions instead of Credentials struct

### External Client Design

* **No factory functions**: Each `New*Client()` handles its own credentials
* **Embedded mocks**: Mock logic inside each client type, not separate classes
* **Return concrete types**: Following Go idioms ("accept interfaces, return structs")
* **YAGNI principle**: Removed unused NotificationClient, RegistryClient, StorageClient interfaces

### Chaos Engineering

* **Generic pod chaos**: Works with any Kubernetes pods, not just admission controllers
* **Convenience builders**: Provide sensible defaults for StackRox components
* **Decoupled functionality**: Chaos logic independent of component-specific code

### gRPC Connections

* **Raw service clients**: Use `v1.New*ServiceClient()` directly, no wrappers
* **Shared connections**: One gRPC connection per suite, multiple service clients
* **HTTP/2 multiplexing**: Efficient concurrent request handling
* **No custom clients package**: Removed `pkg/testutils/clients/` (YAGNI)

## Migration Strategy

This Phase 1 framework enables the three-phase migration approach:

1. **Phase 1 (Complete)**: Framework implementation with all infrastructure
2. **Phase 2 (Next)**: Manual migration of 1-2 representative tests to establish patterns
3. **Phase 3 (Future)**: Sub-agent bulk migration using established patterns and framework

The framework is designed to support both manual migration and AI-assisted conversion with human review.

## Example Usage

See `tests/e2e/ginkgo_example_test.go` for a comprehensive example demonstrating:

* PolicyFieldsTest migration patterns with DescribeTable
* External service integration with automatic mock/real selection
* Chaos engineering for pod resilience testing
* Custom matchers and BDD helpers
* Proper resource cleanup with DeferCleanup
* Raw gRPC service client usage

This example provides the foundation for migrating the remaining ~53 active Groovy test files using established patterns and comprehensive framework infrastructure.

## Files and Structure

```
pkg/testutils/
├── README.md                    # This file
├── env/
│   └── settings.go             # Environment variable management (renamed from credentials/)
├── external/
│   ├── notification.go         # Slack, Email, Webhook, Splunk clients
│   ├── registry.go             # GCR, ECR, ACR, Quay, RedHat clients
│   └── storage.go              # S3, GCS, Azure storage clients
├── chaos/
│   └── pod_chaos.go            # Generic pod chaos testing (renamed from admission_controller.go)
├── ginkgo/
│   └── matchers.go             # Custom Gomega matchers
└── centralgrpc/
    └── connect_to_central.go   # gRPC connection helper

tests/e2e/
└── ginkgo_example_test.go      # Comprehensive example test

Deleted (YAGNI):
├── pkg/testutils/clients/      # Removed - use raw gRPC service clients instead
└── pkg/testutils/ginkgo/suite.go # Removed - referenced deleted packages
```
