---
name: Roxctl CLI
description: "Execute roxctl commands for StackRox/RHACS operations. Auto-applies when working with image scanning, vulnerability checks, security policies, cluster management, init bundles, sensor deployment, or Central API operations."
---

# Roxctl CLI Skill

This skill provides knowledge for executing `roxctl` commands - the CLI for Red Hat Advanced Cluster Security for Kubernetes (RHACS) / StackRox.

## When This Skill Applies

* Image vulnerability scanning and policy checking
* Generating SBOMs (Software Bill of Materials)
* Cluster onboarding with init bundles
* Sensor deployment and management
* Central service administration
* Network policy analysis
* Helm chart operations
* Declarative configuration management

## Connection Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ROX_ENDPOINT` | Central endpoint (host:port) | `localhost:8443` |
| `ROX_SERVER_NAME` | TLS ServerName for SNI | (derived from endpoint) |
| `ROX_API_TOKEN` | API token for authentication | - |
| `ROX_API_TOKEN_FILE` | Path to file containing API token | - |
| `ROX_ADMIN_PASSWORD` | Password for basic auth (admin user) | - |
| `ROX_INSECURE_CLIENT` | Enable insecure connection options | `false` |
| `ROX_INSECURE_CLIENT_SKIP_TLS_VERIFY` | Skip TLS certificate validation | `false` |
| `ROX_CA_CERT_FILE` | Path to custom CA certificate (PEM) | (system certs) |
| `ROX_PLAINTEXT` | Use plaintext (unencrypted) connection | `false` |
| `ROX_DIRECT_GRPC_CLIENT` | Use direct gRPC (no proxy) | `false` |
| `ROX_CLIENT_FORCE_HTTP1` | Force HTTP/1 for all connections | `false` |
| `ROX_USE_KUBECONTEXT` | Connect via kubectl port-forward | `false` |

### Command-Line Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--endpoint` | `-e` | Central endpoint (host:port or URL) |
| `--server-name` | `-s` | TLS ServerName for SNI |
| `--token-file` | | Path to API token file |
| `--password` | `-p` | Admin password for basic auth |
| `--insecure` | | Enable insecure connection options |
| `--insecure-skip-tls-verify` | | Skip TLS certificate validation |
| `--ca` | | Path to custom CA certificate (PEM) |
| `--plaintext` | | Use unencrypted connection |
| `--direct-grpc` | | Use direct gRPC (no proxy) |
| `--force-http1` | | Force HTTP/1 for all connections |
| `--use-current-k8s-context` | | Connect via kubectl port-forward |

**Note:** `--password` and `--token-file` are mutually exclusive.

## Dev Environment Defaults

For this workspace (StackRox deployed in the local KinD cluster):

```bash
export ROX_ENDPOINT="central.stackrox.svc:443"
export ROX_ADMIN_PASSWORD="letmein"
export ROX_INSECURE_CLIENT="true"
```

Or use flags:

```bash
roxctl -e central.stackrox.svc:443 -p letmein --insecure <command>
```

## Authentication Methods

### 1. API Token (Recommended for CI/CD)

```bash
# Via environment variable
export ROX_API_TOKEN="<token-value>"
roxctl central whoami

# Via token file
export ROX_API_TOKEN_FILE="/path/to/token"
roxctl central whoami

# Via flag
roxctl --token-file /path/to/token central whoami
```

### 2. Basic Auth (Username: admin)

```bash
# Via environment variable
export ROX_ADMIN_PASSWORD="letmein"
roxctl central whoami

# Via flag
roxctl -p letmein central whoami
```

### 3. Local Config (Interactive Login)

```bash
# Login and store credentials locally
roxctl central login

# Uses stored access/refresh tokens automatically
roxctl central whoami
```

## Command Reference

### Central Commands

#### Authentication & Identity

```bash
# Interactive login to Central
roxctl central login

# Show current user identity
roxctl central whoami

# Get CA certificate
roxctl central cert
```

#### Init Bundles (Cluster Onboarding)

```bash
# Generate new init bundle
roxctl central init-bundles generate <name> --output-secrets init-bundle.yaml

# List existing init bundles
roxctl central init-bundles list

# Revoke an init bundle
roxctl central init-bundles revoke <id-or-name>
```

#### Database Operations

```bash
# Create Central backup
roxctl central backup

# Restore database
roxctl central db restore <backup-file>
```

#### Debugging

```bash
# Download diagnostic bundle
roxctl central debug download-diagnostics

# Create debug dump
roxctl central debug dump
```

#### Export Operations

```bash
# Export images
roxctl central export images

# Export deployments
roxctl central export deployments

# Export nodes
roxctl central export nodes

# Export pods
roxctl central export pods
```

#### Deployment Generation

```bash
# Generate Central deployment manifests
roxctl central generate k8s none --output-dir ./central-bundle

# Generate with external DB
roxctl central generate k8s pvc --output-dir ./central-bundle \
  --external-db
```

#### PKI & Machine-to-Machine Auth

```bash
# User PKI operations
roxctl central userpki create <name>
roxctl central userpki list
roxctl central userpki delete <id>

# Machine-to-machine token exchange
roxctl central m2m exchange --token <oidc-token>
```

#### Cluster Registration Secrets (CRS)

```bash
# Issue cluster registration secret
roxctl central crs issue <cluster-name>

# List cluster registration secrets
roxctl central crs list

# Revoke cluster registration secret
roxctl central crs revoke <id>
```

### Image Commands

#### Vulnerability Scanning

```bash
# Basic scan (table output)
roxctl image scan --image <registry/image:tag>

# JSON output
roxctl image scan --image <registry/image:tag> --output json

# SARIF output (for GitHub/GitLab integration)
roxctl image scan --image <registry/image:tag> --output sarif

# CSV output
roxctl image scan --image <registry/image:tag> --output csv

# JUnit output (for CI systems)
roxctl image scan --image <registry/image:tag> --output junit

# Force re-scan
roxctl image scan --image <registry/image:tag> --force
```

#### Policy Checking

```bash
# Check image against policies
roxctl image check --image <registry/image:tag>

# JSON output
roxctl image check --image <registry/image:tag> --output json

# SARIF output
roxctl image check --image <registry/image:tag> --output sarif

# Specific policy categories
roxctl image check --image <registry/image:tag> \
  --categories "Vulnerability Management,DevOps Best Practices"
```

#### SBOM Generation

```bash
# CycloneDX format (default)
roxctl image sbom --image <registry/image:tag>

# SPDX format
roxctl image sbom --image <registry/image:tag> --output-format spdx

# Output to file
roxctl image sbom --image <registry/image:tag> --output-file sbom.json
```

### Sensor Commands

```bash
# Generate sensor deployment bundle
roxctl sensor generate k8s --name <cluster-name> \
  --central central.stackrox.svc:443 \
  --output-dir ./sensor-bundle

# Download existing sensor bundle
roxctl sensor get-bundle <cluster-name> --output-dir ./sensor-bundle

# Regenerate sensor certificates
roxctl sensor generate-certs <cluster-name> --output-dir ./certs
```

### Cluster Commands

```bash
# Delete a secured cluster from Central
roxctl cluster delete --name <cluster-name>
```

### Scanner Commands

```bash
# Generate scanner deployment manifests
roxctl scanner generate

# Upload vulnerability database
roxctl scanner upload-db --scanner-db-file <vuln-db.zip>

# Download vulnerability database
roxctl scanner download-db --output-file <vuln-db.zip>
```

### Network Policy Commands

```bash
# Generate network policies from manifests
roxctl netpol generate <path-to-manifests> --output-dir ./netpols

# Analyze connectivity
roxctl netpol connectivity map <path-to-manifests>

# Compare policies (diff)
roxctl netpol connectivity diff <dir1> <dir2>
```

### Helm Commands

```bash
# Output Helm chart
roxctl helm output central-services --output-dir ./helm-chart

# Derive local values from existing deployment
roxctl helm derivelocalvalues <release-name> -n <namespace>
```

### Deployment Commands

```bash
# Check deployment against policies
roxctl deployment check --file deployment.yaml

# JSON output
roxctl deployment check --file deployment.yaml --output json
```

### Declarative Configuration

```bash
# Create declarative config resources
roxctl declarative-config create access-scope <name>
roxctl declarative-config create auth-provider <name>
roxctl declarative-config create notifier <name>
roxctl declarative-config create permission-set <name>
roxctl declarative-config create role <name>

# Lint configuration files
roxctl declarative-config lint --file config.yaml
```

### Utility Commands

```bash
# Display version
roxctl version
roxctl version --json

# Shell completion
roxctl completion bash
roxctl completion zsh
roxctl completion fish
roxctl completion powershell
```

## Common Workflows

### CI/CD Image Scanning

```bash
#!/bin/bash
set -e

export ROX_ENDPOINT="${ROX_CENTRAL_ENDPOINT}"
export ROX_API_TOKEN="${ROX_API_TOKEN}"

IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}"

# Scan for vulnerabilities
roxctl image scan --image "$IMAGE" --output sarif > scan-results.sarif

# Check against policies (fails on violation)
roxctl image check --image "$IMAGE" --output sarif > check-results.sarif
```

### Cluster Onboarding

```bash
#!/bin/bash
CLUSTER_NAME="my-secured-cluster"

# 1. Generate init bundle
roxctl central init-bundles generate "$CLUSTER_NAME" \
  --output-secrets init-bundle.yaml

# 2. Apply init bundle to secured cluster
kubectl apply -f init-bundle.yaml -n stackrox

# 3. Generate sensor bundle
roxctl sensor generate k8s --name "$CLUSTER_NAME" \
  --central central.stackrox.svc:443 \
  --output-dir ./sensor-bundle

# 4. Deploy sensor
kubectl apply -R -f ./sensor-bundle/
```

### SBOM Generation

```bash
# Generate CycloneDX SBOM
roxctl image sbom --image myregistry.io/myapp:v1.0 \
  --output-file sbom-cyclonedx.json

# Generate SPDX SBOM
roxctl image sbom --image myregistry.io/myapp:v1.0 \
  --output-format spdx \
  --output-file sbom-spdx.json
```

### Network Policy Analysis

```bash
# Generate recommended network policies
roxctl netpol generate ./k8s-manifests --output-dir ./generated-policies

# Analyze connectivity between workloads
roxctl netpol connectivity map ./k8s-manifests

# Compare before/after policies
roxctl netpol connectivity diff ./policies-before ./policies-after
```

## Output Formats

| Command | Formats |
|---------|---------|
| `image scan` | `table`, `json`, `csv`, `sarif`, `junit` |
| `image check` | `table`, `json`, `sarif`, `junit` |
| `image sbom` | `cyclonedx`, `spdx` |
| `deployment check` | `table`, `json` |
| `central export *` | `json` |

## Error Handling

### Connection Errors

**Error:** `connection refused` or `no route to host`
```bash
# Verify endpoint is reachable
curl -k https://central.stackrox.svc:443/v1/ping

# Try with port-forwarding
roxctl --use-current-k8s-context central whoami
```

**Error:** `x509: certificate signed by unknown authority`
```bash
# Option 1: Skip TLS verification (development only)
roxctl --insecure-skip-tls-verify central whoami

# Option 2: Provide CA certificate
roxctl --ca /path/to/ca.pem central whoami
```

### Authentication Errors

**Error:** `unauthenticated` or `401`
```bash
# Verify credentials
echo $ROX_API_TOKEN
echo $ROX_ADMIN_PASSWORD

# Test with explicit credentials
roxctl -p letmein --insecure -e central.stackrox.svc:443 central whoami
```

**Error:** `token expired`
```bash
# Re-login to refresh tokens
roxctl central login
```

### Policy Violations

When `roxctl image check` fails with policy violations:
* Exit code 1 indicates policy violations
* Use `--output json` to parse violation details
* Use `--fail-on-unfixable-violations=false` to ignore unfixable CVEs
