# StackRox/RHACS Feature Documentation

This directory contains feature-oriented documentation for StackRox/RHACS. Each document answers: **"I need to improve feature X — where is the code and how does it work?"**

These docs are technical guides for developers working on the codebase, covering architecture, key code locations, common tasks, and recent changes.

---

## Feature Documentation Index

### 15. Authentication and Authorization
**File**: [authentication.md](authentication.md)
**Packages**: `pkg/auth`, `central/auth`, `central/apitoken`
**Lines**: 280

**What it covers**:
- Multi-provider authentication framework (OIDC, SAML, OpenShift OAuth, User PKI, Basic Auth)
- JWT token issuance and validation with RSA-256 signatures
- Role mapping from external groups to StackRox roles
- API tokens for CI/CD and automation
- Machine-to-machine (M2M) authentication for GitHub Actions, Kubernetes ServiceAccounts, generic OIDC
- Token lifecycle management (issuance, validation, revocation, expiration)

**Key use cases**:
- Adding new authentication provider types
- Configuring role mapping for external identity providers
- Setting up M2M authentication for automation
- Managing API tokens programmatically
- Understanding token validation and security

**Primary files**:
- `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/auth/authproviders/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/auth/tokens/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/central/auth/m2m/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/central/apitoken/`

---

### 16. roxctl CLI
**File**: [roxctl-cli.md](roxctl-cli.md)
**Package**: `roxctl/`
**Lines**: 270

**What it covers**:
- Command-line interface architecture and command tree
- Administrative commands (cluster management, database operations, backups, certificates)
- CI/CD integration patterns (image and deployment policy checking)
- Security scanning commands (image vulnerability scanning, SBOM generation)
- Deployment generation (YAML manifests for Central, Sensor, Scanner)
- Authentication methods (API tokens, basic auth, interactive login, M2M exchange)
- Output formats (table, JSON, JUnit, SARIF) for CI/CD integration

**Key use cases**:
- Integrating StackRox into CI/CD pipelines
- Deploying Central and Sensor via generated manifests
- Scanning images and deployments for policy violations
- Managing clusters and init bundles
- Debugging Central and Sensor deployments
- Implementing new roxctl commands

**Primary files**:
- `/Users/rc/go/src/github.com/stackrox/stackrox/roxctl/main.go`
- `/Users/rc/go/src/github.com/stackrox/stackrox/roxctl/maincommand/command.go`
- `/Users/rc/go/src/github.com/stackrox/stackrox/roxctl/central/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/roxctl/sensor/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/roxctl/image/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/roxctl/deployment/`

---

### 17. Helm Configuration and Meta-Templating
**File**: [helm-configuration.md](helm-configuration.md)
**Packages**: `pkg/helm`, `image/templates/helm`
**Lines**: 300

**What it covers**:
- Two-stage templating system (meta-templating + Helm rendering)
- MetaValues structure for build-time configuration
- Chart instantiation code paths (roxctl, operator, Central)
- stackrox-central-services chart (Central, DB, Scanner, Scanner-v4)
- stackrox-secured-cluster-services chart (Sensor, Collector, Admission Controller)
- Multi-stage defaults system for secured cluster
- Configuration shape, defaults, and expandables
- Common customizations (resources, external DB, Scanner-v4, admission control)

**Key use cases**:
- Adding new Helm chart values
- Customizing deployments for different environments
- Understanding chart generation and rendering
- Modifying chart templates
- Testing chart changes
- Creating platform-specific defaults

**Primary files**:
- `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/helm/charts/meta.go`
- `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/helm/template/chart_template.go`
- `/Users/rc/go/src/github.com/stackrox/stackrox/image/templates/helm/stackrox-central/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/image/templates/helm/stackrox-secured-cluster/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/image/templates/helm/shared/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/image/embed_charts.go`

---

### 18. Operator Deployment and Management
**File**: [operator-deployment.md](operator-deployment.md)
**Package**: `operator/`
**Lines**: 230

**What it covers**:
- Kubernetes operator architecture built on Kubebuilder
- Custom Resource Definitions (Central, SecuredCluster)
- Reconciliation engine and lifecycle management
- Defaulting mechanism for upgrade scenarios
- Helm values translation (CRD → Helm chart values)
- Extensions system (CA rotation, password generation)
- Upgrade strategy with version detection
- CA certificate rotation (automatic and manual)

**Key use cases**:
- Adding new CRD fields
- Implementing reconciliation logic
- Creating operator extensions
- Managing Central and SecuredCluster deployments
- Understanding operator upgrade process
- Debugging reconciliation issues

**Primary files**:
- `/Users/rc/go/src/github.com/stackrox/stackrox/operator/apis/platform/v1alpha1/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/operator/pkg/central/reconciliation/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/operator/pkg/securedcluster/reconciliation/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/operator/pkg/defaults/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/operator/pkg/extensions/`

---

### 19. Image Registry Integration
**File**: [image-registry-integration.md](image-registry-integration.md)
**Packages**: `pkg/registries`, `central/imageintegration`
**Lines**: 280

**What it covers**:
- Unified registry abstraction for 11 registry types
- Cloud provider registries (ECR, GCR/GAR, ACR) with automatic token refresh
- Enterprise registries (Quay, Artifactory, Nexus)
- Generic Docker V2 and specialized registries (GHCR, IBM, Red Hat)
- Factory pattern for registry creation
- Registry matching and priority order
- Metadata retrieval (manifests, layers, Dockerfile instructions)
- Multi-architecture image support
- Tag listing with pagination
- Delegated scanning (Sensor-side registry access)
- Auto-generated integrations (global pull secret, namespace pull secrets)

**Key use cases**:
- Adding support for new registry types
- Configuring registry authentication
- Understanding registry matching logic
- Implementing token refresh for cloud providers
- Debugging registry connectivity issues
- Working with multi-architecture images

**Primary files**:
- `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/registries/factory_impl.go`
- `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/registries/set_impl.go`
- `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/registries/types/types.go`
- `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/registries/docker/docker.go`
- `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/registries/ecr/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/registries/google/`
- `/Users/rc/go/src/github.com/stackrox/stackrox/central/imageintegration/`

---

## How to Use These Docs

### For New Developers

Start with the feature you're working on to understand:
- **Architecture**: How the feature is organized
- **Key Code Locations**: Where to find implementations
- **Common Tasks**: Step-by-step guides for common modifications

### For Feature Development

1. Read the relevant feature doc to understand current implementation
2. Locate key files and interfaces
3. Review recent changes to understand evolution
4. Follow common task examples for modifications
5. Check related components for integration points

### For Debugging

Use these docs to:
- Understand component interactions
- Locate logging and metrics instrumentation
- Find configuration options and environment variables
- Identify known limitations and technical debt

### For Code Review

Reference these docs to:
- Verify architectural patterns are followed
- Check that changes align with recent evolution
- Ensure integration points are considered
- Validate against known limitations

---

## Additional Documentation

### Architecture Documentation
- **pkg/**: `/Users/rc/go/src/github.com/stackrox/stackrox/doc/pkg/`
- **central/**: `/Users/rc/go/src/github.com/stackrox/stackrox/doc/central/`
- **operator/**: `/Users/rc/go/src/github.com/stackrox/stackrox/doc/operator/`

### Development Guides
- **Operator CRD Extension**: `/Users/rc/go/src/github.com/stackrox/stackrox/operator/EXTENDING_CRDS.md`
- **Operator Defaulting**: `/Users/rc/go/src/github.com/stackrox/stackrox/operator/DEFAULTING.md`
- **Helm Chart Templating**: `/Users/rc/go/src/github.com/stackrox/stackrox/image/templates/CHART_TEMPLATING.md`
- **Changing Helm Charts**: `/Users/rc/go/src/github.com/stackrox/stackrox/image/templates/CHANGING_CHARTS.md`
- **Working with Helm**: `/Users/rc/go/src/github.com/stackrox/stackrox/image/templates/README.md`

### Testing
- **Helm Test Framework**: `pkg/helm/charts/testutils/`
- **Integration Tests**: `/Users/rc/go/src/github.com/stackrox/stackrox/qa-tests-backend/`

---

## Contributing

When adding new features or making significant changes:

1. **Update the relevant feature doc** with new code locations and patterns
2. **Document breaking changes** and migration paths
3. **Add common task examples** for new functionality
4. **Update the index** if adding entirely new features

---

## Feedback

If you find these docs helpful or have suggestions for improvement, please provide feedback to the Platform Security Team or Documentation Team.

**Last Updated**: March 2026
