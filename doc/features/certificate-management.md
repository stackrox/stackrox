# Certificate Management

StackRox uses mutual TLS (mTLS) to secure all service-to-service communication. The system manages a dedicated certificate authority, issues certificates for each component, supports rotation without downtime, and monitors certificate health.

**Primary Packages**: `pkg/mtls`, `pkg/certgen`, `central/credentialexpiry`

## What It Does

All StackRox components communicate over mTLS using certificates from a StackRox-specific service CA. The system provides:

- 5-year service CA with automatic generation
- Per-component certificates (Central, Sensor, Scanner, etc.) with 365-day lifetime
- Dual CA support enabling zero-downtime rotation
- Hot-reload of certificates without service restart
- Expiration monitoring with health status reporting
- Init bundles for cluster onboarding with sensor credentials

## Architecture

### Service Identity

Each StackRox component has a unique subject combining service type, identifier, and optional init bundle ID. The `pkg/mtls/cn.go` Subject type encodes identity into X.509 common names using the pattern `service-type:identifier[:init-bundle-id]`.

Predefined subjects exist for all components: Central, Sensor, Admission Control, Scanner (legacy and V4 variants), and databases. Identity extends subjects with serial numbers and validity periods defined in `pkg/mtls/crypto.go`.

### CA Management

The CA interface in `pkg/mtls/ca.go` provides certificate issuance, validation, and property checking. CAs lazily load from filesystem on first access and cache for subsequent calls.

**Dual CA Support**: During rotation, two CAs are trusted simultaneously:
- Primary: `/run/secrets/stackrox.io/certs/ca.pem`
- Secondary: `/run/secrets/stackrox.io/certs/ca-secondary.pem`

New certificates issue from primary while validation accepts both, enabling gradual rollout.

### Certificate Issuance

The `pkg/mtls/crypto.go` module generates certificates through a standard flow: validate subject, generate 64-bit random serial, create CSR, determine hostnames, sign with CA private key, return PEM-encoded cert and key.

**Certificate Profiles**:

| Profile | Lifetime | Backdate | Usage |
|---------|----------|----------|-------|
| Default | 365 days | 1 hour | Service certificates |
| Ephemeral (Hours) | 3 hours | None | Non-revocable init bundles |
| Ephemeral (Days) | 2 days | None | Short-lived credentials |
| CRS Profile | 24 hours | 1 hour | Registration service |

Clock skew tolerance uses 1-hour backdate on NotBefore, accounting for time differences between nodes.

### Certificate Generation

The `pkg/certgen/` package provides functions for generating CAs and issuing service certificates. Core operations include `GenerateCA()` for new 5-year CAs, per-service issuance functions like `IssueCentralCert()`, and rotation helpers `PromoteSecondaryCA()` and `RemoveSecondaryCA()`.

File map conventions use consistent naming: `ca-cert.pem` and `ca-key.pem` for primary CA, `ca-secondary.pem` and `ca-secondary-key.pem` during rotation, and `[prefix]cert.pem`/`[prefix]key.pem` for services.

### Hot-Reload

The `pkg/mtls/certwatch/` system monitors certificate directories for changes. TLSConfigHolder in `certwatch/tls_config_holder.go` maintains atomic pointers to live TLS configs, updated automatically when files change. The watcher uses `k8scfgwatch` to detect filesystem events and reload certificates without service restart.

### Expiry Monitoring

Central's `credentialexpiry/` service tracks expiration across all components: Central certificates, Scanner variants, registry integrations, and database connections. The service interface provides methods to query expiry dates and component-specific status, with graceful degradation when certificates are temporarily unavailable.

## CA Rotation

### Rotation Process

Rotation follows a five-step process:

1. **Generate Secondary**: Create new CA and add to fileMap
2. **Issue Certificates**: Sign new service certs with secondary CA
3. **Deploy Dual CAs**: Both primary and secondary trusted cluster-wide
4. **Promote**: Swap secondary to primary position
5. **Remove Old**: Delete original CA after grace period

The Operator automates this flow, detecting approaching expiration and orchestrating rotation across all pods. Recent work in ROX-27962 and ROX-27963 added Operator automation and Central dual-CA support respectively.

## Init Bundles

Init bundles package certificates and configuration for Sensor clusters. Components include service CA certificate, Sensor client certificate signed by CA, Sensor private key, and Central endpoint address.

Generation occurs via `roxctl central init-bundles generate` or the API service in `central/clusterinit/backend/`. Bundles produce Kubernetes Secrets with base64-encoded certificates.

**Bundle vs CRS**:

| Aspect | Init Bundle | Cluster Registration Secret |
|--------|-------------|----------------------------|
| Certificate | Static, embedded | Generated on-demand |
| Lifetime | Long-lived (no expiry) | Time-limited (24h default) |
| Cluster Count | One per cluster | Multiple (configurable max) |
| Use Case | Production | Ephemeral/automation |

Recent updates in ROX-27238 made CRS expiration configurable with 24-hour default.

## Certificate Lifecycle

### Creation and Storage

Service certificates use 4096-bit RSA keys with CFSSL library for signing. Serial numbers are 64-bit cryptographically random values, ensuring negligible collision probability.

Storage typically uses Kubernetes Secrets with type `kubernetes.io/tls`, containing ca.pem, cert.pem, and key.pem. Alternative file mounts place certificates in `/run/secrets/stackrox.io/certs/` with consistent naming.

### Security Properties

**CA Certificate**: Self-signed 4096-bit RSA root with 5-year validity and Basic Constraints CA=true, pathlen=0 preventing intermediate CAs.

**Service Certificate**: Dual-use client+server authentication, 4096-bit RSA, 365-day validity, with SANs covering all service hostnames.

**mTLS Benefits**: Prevents MITM attacks, provides mutual authentication, encrypts all traffic, and verifies service identity at connection time.

## Configuration

### Environment Variables

```
ROX_MTLS_CA_FILE               # CA certificate path
ROX_MTLS_CA_KEY_FILE           # CA private key path
ROX_MTLS_SECONDARY_CA_FILE     # Secondary CA (rotation)
ROX_MTLS_SECONDARY_CA_KEY_FILE # Secondary key
ROX_MTLS_CERT_FILE             # Service certificate
ROX_MTLS_KEY_FILE              # Service private key
```

Default location: `/run/secrets/stackrox.io/certs/`

### Best Practices

- Plan CA rotation 6+ months before expiration
- Backup CA private key securely offline for disaster recovery
- Enable automatic monitoring alerts at 30, 14, and 7 days before expiration
- Test rotation in non-production environments first
- Generate separate init bundles per production cluster
- Use CRS for temporary/development clusters only
- Limit CRS max_registrations to expected count plus buffer

## Implementation

**Core**: `pkg/mtls/ca.go`, `pkg/mtls/cn.go`, `pkg/mtls/crypto.go`
**Watching**: `pkg/mtls/certwatch/certwatch.go`, `pkg/mtls/certwatch/tls_config_holder.go`
**Generation**: `pkg/certgen/ca.go`, `pkg/certgen/service_certs.go`, `pkg/certgen/rotation.go`
**Monitoring**: `central/credentialexpiry/service_impl.go`
**Init Bundles**: `central/clusterinit/backend/backend_impl.go`
