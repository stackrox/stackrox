# Roxctl Image Scanning Reference

## Overview

`roxctl image scan` and `roxctl image check` are the primary commands for CI/CD integration, providing vulnerability detection and policy enforcement.

## Image Scan

Scans an image for vulnerabilities and reports findings.

### Basic Usage

```bash
roxctl image scan --image <registry/image:tag>
```

### Output Formats

| Format | Flag | Use Case |
|--------|------|----------|
| Table | `--output table` | Human-readable (default) |
| JSON | `--output json` | Programmatic parsing |
| CSV | `--output csv` | Spreadsheet import |
| SARIF | `--output sarif` | GitHub/GitLab code scanning |
| JUnit | `--output junit` | CI test reporting |

### Scan Options

```bash
# Force re-scan (ignore cache)
roxctl image scan --image nginx:latest --force

# Include base image vulnerabilities
roxctl image scan --image myapp:v1 --include-snoozed

# Specify registry credentials
roxctl image scan --image private.registry.io/app:v1 \
  --registry-username user \
  --registry-password pass
```

### SARIF Integration

SARIF (Static Analysis Results Interchange Format) integrates with:
* GitHub Code Scanning
* GitLab SAST
* Azure DevOps
* VS Code SARIF Viewer

```bash
# Generate SARIF report
roxctl image scan --image myapp:v1 --output sarif > results.sarif

# GitHub Actions upload
- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v2
  with:
    sarif_file: results.sarif
```

## Image Check

Checks an image against configured security policies.

### Basic Usage

```bash
roxctl image check --image <registry/image:tag>
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No policy violations |
| 1 | Policy violations detected |
| 2 | Error (connection, auth, etc.) |

### Check Options

```bash
# Specific policy categories
roxctl image check --image nginx:latest \
  --categories "Vulnerability Management"

# Ignore unfixable vulnerabilities
roxctl image check --image nginx:latest \
  --fail-on-unfixable-violations=false

# Output formats
roxctl image check --image nginx:latest --output json
roxctl image check --image nginx:latest --output sarif
```

### Policy Categories

* `Anomalous Activity`
* `DevOps Best Practices`
* `Docker CIS`
* `Kubernetes`
* `Kubernetes Events`
* `Network Tools`
* `Package Management`
* `Privileges`
* `Security Best Practices`
* `Supply Chain Security`
* `System Modification`
* `Vulnerability Management`

## SBOM Generation

Generate Software Bill of Materials (SBOM) for an image.

### Formats

```bash
# CycloneDX (default)
roxctl image sbom --image nginx:latest

# SPDX
roxctl image sbom --image nginx:latest --output-format spdx
```

### Output Options

```bash
# Write to file
roxctl image sbom --image nginx:latest --output-file sbom.json

# Specify CycloneDX version
roxctl image sbom --image nginx:latest --cyclonedx-version 1.4
```

## CI/CD Pipeline Examples

### GitHub Actions

```yaml
name: Security Scan

on: [push, pull_request]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Build Image
        run: docker build -t myapp:${{ github.sha }} .

      - name: Push to Registry
        run: |
          docker tag myapp:${{ github.sha }} registry.example.com/myapp:${{ github.sha }}
          docker push registry.example.com/myapp:${{ github.sha }}

      - name: Scan Image
        env:
          ROX_ENDPOINT: ${{ secrets.ROX_ENDPOINT }}
          ROX_API_TOKEN: ${{ secrets.ROX_API_TOKEN }}
        run: |
          roxctl image scan \
            --image registry.example.com/myapp:${{ github.sha }} \
            --output sarif > scan-results.sarif

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: scan-results.sarif

      - name: Check Policies
        env:
          ROX_ENDPOINT: ${{ secrets.ROX_ENDPOINT }}
          ROX_API_TOKEN: ${{ secrets.ROX_API_TOKEN }}
        run: |
          roxctl image check \
            --image registry.example.com/myapp:${{ github.sha }}
```

### GitLab CI

```yaml
stages:
  - build
  - security

variables:
  IMAGE: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA

build:
  stage: build
  script:
    - docker build -t $IMAGE .
    - docker push $IMAGE

security-scan:
  stage: security
  variables:
    ROX_ENDPOINT: $ROX_CENTRAL_ENDPOINT
    ROX_API_TOKEN: $ROX_API_TOKEN
  script:
    - roxctl image scan --image $IMAGE --output json > scan.json
    - roxctl image check --image $IMAGE
  artifacts:
    reports:
      sast: scan.json
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any

    environment {
        ROX_ENDPOINT = credentials('rox-endpoint')
        ROX_API_TOKEN = credentials('rox-api-token')
        IMAGE = "${env.REGISTRY}/${env.IMAGE_NAME}:${env.BUILD_NUMBER}"
    }

    stages {
        stage('Build') {
            steps {
                sh 'docker build -t ${IMAGE} .'
                sh 'docker push ${IMAGE}'
            }
        }

        stage('Security Scan') {
            steps {
                sh '''
                    roxctl image scan --image ${IMAGE} --output json > scan-results.json
                    roxctl image check --image ${IMAGE}
                '''
            }
            post {
                always {
                    archiveArtifacts artifacts: 'scan-results.json'
                }
            }
        }
    }
}
```

## Interpreting Results

### Vulnerability Severities

| Severity | CVSS Score | Action |
|----------|------------|--------|
| Critical | 9.0 - 10.0 | Immediate remediation |
| Important/High | 7.0 - 8.9 | Prioritize fix |
| Moderate/Medium | 4.0 - 6.9 | Schedule fix |
| Low | 0.1 - 3.9 | Monitor |

### Policy Violation Actions

| Enforcement | Effect |
|-------------|--------|
| Inform | Warning only, no build failure |
| Fail | Build/deployment fails |

### JSON Output Fields

```json
{
  "result": {
    "summary": {
      "CRITICAL": 2,
      "IMPORTANT": 5,
      "MODERATE": 12,
      "LOW": 8
    },
    "vulnerabilities": [
      {
        "cve": "CVE-2024-1234",
        "severity": "CRITICAL",
        "cvss": 9.8,
        "component": {
          "name": "openssl",
          "version": "1.1.1k"
        },
        "fixedBy": "1.1.1l"
      }
    ]
  }
}
```

## Best Practices

1. **Scan on Build**: Integrate scanning into build pipeline before deployment
2. **Block on Critical**: Fail builds with critical vulnerabilities
3. **Base Image Selection**: Use verified, minimal base images
4. **Regular Rescans**: Periodically rescan deployed images for new CVEs
5. **SBOM Archival**: Store SBOMs for compliance and incident response
6. **Policy Tuning**: Adjust policies to reduce false positives
7. **Fix Forward**: Update dependencies rather than suppressing alerts
