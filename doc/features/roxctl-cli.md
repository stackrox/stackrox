# roxctl CLI

Official command-line interface for administrative tasks, CI/CD integration, security scanning, deployment generation, and network policy management.

**Primary Package**: `roxctl/`
**Framework**: spf13/cobra

## What It Does

The `roxctl` tool provides comprehensive CLI access to StackRox/RHACS functionality: administrative tasks (cluster management, database operations, backups, certificates), CI/CD integration (image and deployment policy checking), security scanning (vulnerability scanning, SBOM generation), deployment generation (YAML manifests for Central, Sensor, Scanner), network policy management, and automation via API client.

Command structure follows `roxctl <noun> [subcommands...]` pattern per ADR-0004.

## Architecture

### Design Principles

1. Command structure: `roxctl <noun> [subcommands...]`
2. Environment abstraction: `environment.Environment` interface for dependency injection
3. Testability: Separable business logic for unit testing
4. Output flexibility: Multiple formats (table, JSON, JUnit, SARIF) via `printer.ObjectPrinter`

Entry point in `roxctl/main.go` sets user agent, registers devbuild settings, patches persistent pre-run hooks for command path tracking, and applies custom help formatting via `utils.FormatHelp`.

The `roxctl/maincommand/command.go` registers all command hierarchies.

### Environment Interface

Defined in `roxctl/common/environment/environment.go`, provides connection management (GRPCConnection(), HTTPClient()), I/O streams (InputOutput(), Logger(), ColorWriter()), and configuration (GetConfig(), SetConfig()).

Benefits: mockable for testing, centralizes gRPC connection setup, consistent HTTP client creation, unified logging and output handling.

## Command Tree

Major command groups:

**central**: Service management including backup (database and certificates), cert (download chain), crs (Cluster Registration Secrets, feature-flagged), db (backup/restore/transfer), debug (authz-trace, diagnostics, logs, db stats), export (deployments/images/nodes/pods, tech preview), generate (Central deployment YAML), init-bundles (management), login (interactive auth), m2m (machine-to-machine auth), userpki (certificate management), whoami (auth context).

**cluster**: Operations including delete.

**collector**: Support package upload.

**completion**: Shell completion scripts (bash, zsh, fish, PowerShell).

**declarative-config**: Feature-flagged configuration with create (templates for access-scope, auth-provider, notifier, etc.) and lint (validation).

**deployment**: Policy checking with check command.

**helm**: Chart operations including output (values) and derive-local-values (from cluster).

**image**: Scanning including check (build-time policies), scan (vulnerabilities), sbom (generation).

**netpol**: Network policy operations including connectivity (map, diff) and generate (deprecated).

**scanner**: Service deployment including generate (YAML), upload-db, download-db.

**sensor**: Deployment management including generate (k8s/openshift), get-bundle (existing cluster), generate-certs.

**version**: Display version information.

## Key Commands

### central generate

Generates Central deployment YAML for Kubernetes or OpenShift. Located in `roxctl/central/generate/generate.go`. Supports interactive mode with prompts, external volume configuration, license integration, and custom certificate handling.

### central init-bundles

Manages cluster init bundles (certificates for sensor authentication). Located in `roxctl/central/initbundles/`. Commands: generate (create new bundle), list (show all bundles), revoke (invalidate bundle), fetch-ca (get CA config for Helm).

### central login

Interactive authentication and configuration storage in `roxctl/central/login/login.go`. Features browser-based OAuth flow, credential storage in config file (~/.roxctl/config), automatic endpoint validation, and TLS certificate verification.

### central m2m exchange

Exchanges service account token for API token in `roxctl/central/m2m/`. Use case: Kubernetes ServiceAccount → StackRox API token for in-cluster automation.

### deployment check

Checks deployments against deploy-time policies for CI/CD integration. Located in `roxctl/deployment/check/check.go`.

Key flags: -f/--file (YAML files, required, repeatable), -c/--categories (policy categories, comma-separated), -r/--retries (default: 3), -d/--retry-delay (default: 3 seconds), --force (bypass cache), --cluster (context), -n/--namespace (default: "default"), --output (table/json/junit/sarif), --timeout (default: 10 minutes).

Exit codes: 0 (no violations or non-enforcing only), 1 (at least one enforcing policy violated).

### image check

Checks image against build-time policies in `roxctl/image/check/check.go`. Key flags: -i/--image (required), --send-notifications, -c/--categories, --output, --force, --cluster, --namespace, --timeout.

### image scan

Scans image for vulnerabilities in `roxctl/image/scan/scan.go`. Key flags: --include-snoozed, --output (table/json/csv/sarif).

### image sbom

Generates Software Bill of Materials in `roxctl/image/sbom/sbom.go`. Outputs CycloneDX format.

### sensor generate

Generates Sensor deployment files for secured clusters in `roxctl/sensor/generate/generate.go`.

Key flags: --name (required), --central (default: central.stackrox:443), --output-dir, --main-image-repository, --collector-image-repository, --collection-method (none/default/core_bpf), --create-upgrader-sa (default: true), --istio-support, --continue-if-exists.

Deprecated admission controller flags (as of 4.9): --admission-controller-listen-on-creates/updates, --admission-controller-scan-inline, --admission-controller-enforce-on-creates/updates, --admission-controller-timeout.

## Authentication

Priority order: API Token (flag or env), Basic Authentication (flag or env), Interactive Login (stored config), M2M Token Exchange.

**API Token** (recommended for CI/CD): Via `--token-file` flag or `ROX_API_TOKEN_FILE`/`ROX_API_TOKEN` environment. Implementation in `roxctl/common/auth/token.go`.

**Basic Authentication**: Via `--password` flag or `ROX_ADMIN_PASSWORD` environment. Implementation in `roxctl/common/auth/basic.go`.

## Connection Configuration

**Endpoint**: Via `--endpoint` flag or `ROX_ENDPOINT` environment (default: localhost:8443).

**TLS Configuration**:
- Server Name (SNI): `--server-name` or `ROX_SERVER_NAME`
- Custom CA: `--ca` or `ROX_CA_CERT_FILE`
- Skip TLS verification: `--insecure-skip-tls-verify` or `ROX_INSECURE_CLIENT_SKIP_TLS_VERIFY`
- Insecure plaintext: `--insecure --plaintext` or `ROX_INSECURE_CLIENT` + `ROX_PLAINTEXT`

**Advanced Options**:
- Force HTTP/1: `--force-http1` or `ROX_CLIENT_FORCE_HTTP1`
- Direct gRPC: `--direct-grpc` or `ROX_DIRECT_GRPC_CLIENT`
- Port forwarding: `--use-current-k8s-context` or `ROX_USE_CURRENT_K8S_CONTEXT`

Implementation in `roxctl/common/flags/endpoint.go`.

## Output Formats

**Table**: Human-readable default format.

**JSON**: Structured output (new format not backward compatible with deprecated `--json` flag).

**JUnit**: JUnit XML for CI/CD integration. Test cases represent policies, failed cases are enforcing violations, skipped cases are non-enforcing violations, error messages contain violation details.

**SARIF**: Static Analysis Results Interchange Format for IDE and security tool integration (GitHub Advanced Security). Includes policy violations as rules, severity mapping, remediation guidance, and violation details.

**CSV**: Comma-separated values for `image scan`.

## CI/CD Integration

**GitHub Actions**: Download roxctl, check image, upload SARIF to code scanning.

**GitLab CI**: Run security scan, generate JUnit report as artifact.

**Jenkins Pipeline**: Archive scan results as JSON.

**Best Practices**:
1. Use API tokens (store as secrets, not passwords)
2. Set timeouts (`--timeout` for slow networks/large images)
3. Enable retries (`--retries` for transient issues)
4. Output format (junit/sarif for native CI/CD integration)
5. Cache management (use `--force` sparingly)
6. Error handling (check exit codes to fail builds)
7. Context information (provide `--cluster` and `--namespace` for accurate evaluation)

## Environment Variables

**Authentication**: `ROX_API_TOKEN`, `ROX_API_TOKEN_FILE`, `ROX_ADMIN_PASSWORD`

**Connection**: `ROX_ENDPOINT`, `ROX_SERVER_NAME`, `ROX_CA_CERT_FILE`, `ROX_INSECURE_CLIENT`, `ROX_INSECURE_CLIENT_SKIP_TLS_VERIFY`, `ROX_PLAINTEXT`, `ROX_DIRECT_GRPC_CLIENT`, `ROX_CLIENT_FORCE_HTTP1`, `ROX_USE_CURRENT_K8S_CONTEXT`

**Output**: `ROX_NO_COLOR`

**Features**: `ROX_DECLARATIVE_CONFIGURATION`, `ROX_CLUSTER_REGISTRATION_SECRETS`

## Development

Command structure requirements: follow noun-based pattern, use environment.Environment, implement RunE (never Run), bind flags to struct fields, separate Construct(), Validate(), and business logic methods.

Breaking changes need intrinsic value (functionality/stability/usability/maintenance), deprecation notice in CHANGELOG.md, aliases for renamed commands/flags, and backward compatibility consideration.

Deprecation: Mark flags deprecated via `utils.Must(c.Flags().MarkDeprecated("old-flag", "message"))`, hide commands with `Hidden: true`, add warnings in RunE, redirect to new commands.

## Recent Changes

Recent work addressed ROX-33178 (image deployment handling), ROX-26769 (CRS max-registrations), ROX-31431 (improved auth error messages), ROX-31432 (better endpoint error handling), ROX-32851 (deprecated netpol NP-Guard commands), ROX-31296 (updated to non-deprecated K8s fake clientset), ROX-31393 (path traversal prevention with os.Root), ROX-30937 (baseline auto-locking config), ROX-27920 (re-introduced cluster/namespace flags for SBOM), ROX-30727 (ignore deprecated admission controller flags), ROX-3024 (gRPC retry on deadline), ROX-24956 (default admission timeout 0), ROX-30034/29995/29996 (new admission options), ROX-28673 (refactored TLS config), and Helm upgrade 3.18.6 → 3.19.2.

## Implementation

**Main**: `roxctl/main.go`, `roxctl/maincommand/command.go`
**Commands**: `roxctl/central/`, `roxctl/deployment/`, `roxctl/image/`, `roxctl/sensor/`, `roxctl/helm/`
**Common**: `roxctl/common/auth/`, `roxctl/common/flags/`, `roxctl/common/environment/`
**Testing**: Use testify suite pattern with mocked Environment in `roxctl/common/environment/mocks/`
