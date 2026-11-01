# TLS certificates in StackRox

This document covers TLS certificate management in StackRox, primarily the internal CA and mTLS.

> **IMPORTANT — Rules for new TLS code:**
>
> 1. Changes that affect the TLS architecture MUST be reflected in this README.
> 2. New servers and client connections MUST hot-reload certificates (use
>    `certwatch` or `GetCertificate`/`GetClientCertificate` callbacks). Do NOT
>    add process-lifetime caches for leaf certs.

## Architecture

- StackRox uses mTLS for most inter-service communication. Service type is extracted
  from the client TLS certificate for RPC authorization.
- Exceptions (TLS but not mTLS): Postgres connections, config-controller.
- Internal CA: self-signed, 5-year validity, stored in `central-tls` secret.
  Only Central and the Operator have access to the CA private key.
- Leaf certs: 1-year validity, per-service secrets. SANs match service DNS names.
  - Central side: `<service>-tls` (e.g. `central-tls`, `scanner-tls`, `central-db-tls`)
  - Secured Cluster side: `tls-cert-<service>` (e.g. `tls-cert-sensor`, `tls-cert-collector`)
  - Legacy (init bundles, deprecated): `<service>-tls` on Secured Cluster side too (e.g. `sensor-tls`)
- Custom default cert: user-provided cert for Central's external-facing endpoints.

## Key file paths

- `CertsPrefix` = `/run/secrets/stackrox.io/certs/` (`pkg/mtls/crypto.go`)
- CA: `CertsPrefix/ca.pem`, `CertsPrefix/ca-key.pem`
- Secondary CA (during rotation): `CertsPrefix/ca-secondary.pem`, `CertsPrefix/ca-secondary-key.pem`
- Leaf cert: `CertsPrefix/cert.pem`, `CertsPrefix/key.pem`
- Default cert: `/run/secrets/stackrox.io/default-tls-cert/` (`tls.crt`/`tls.key`)
- OpenShift service-serving cert: `/run/secrets/stackrox.io/ocp-tls/` (`tls.crt`/`tls.key`) — issued by OpenShift's service CA for the `central-ocp` Service

## Key code locations

- Service identity extraction from client cert CN: `pkg/mtls/cn.go`, `pkg/grpc/authn/service/extractor.go`
- CA loading (sync.Once, process-lifetime cache): `pkg/mtls/crypto.go`
- Leaf cert loading (no cache, reads disk each call): `mtls.LeafCertificateFromFile()` in `pkg/mtls/crypto.go`
- Cert file watcher: `pkg/mtls/certwatch/certwatch.go` — polls directory every 5s, debounces, calls update callback
- TLS config composition: `pkg/mtls/certwatch/tls_config_holder.go` — `atomic.Pointer[tls.Config]`, rotates session ticket keys on every update to invalidate cached TLS sessions
- Trust pool builders: `pkg/mtls/verifier/verify.go` — `TrustedCertPool()`, `NonCA.TLSConfig()`
- Central TLS manager: `central/tlsconfig/manager_impl.go` — composes server certs + trust roots for incoming connections
- Central TLS cert loaders: `central/tlsconfig/tlsconfig.go` — `LoadInternalCertificateFromDirectory()`, `MaybeGetDefaultTLSCertificateFromDirectory()`, `MaybeLoadOpenShiftTLSCertificateFromDirectory()`
- OpenShift service-serving TLS: `central/tlsconfig/openshift_tls.go` — watches `/run/secrets/stackrox.io/ocp-tls/` for certs issued by OpenShift's service CA for the `central-ocp` Service
- TLS challenge endpoint: `central/metadata/service/service_impl.go`
- Cert issuance for Secured Clusters: `central/securedclustercertgen/certificates.go`
- CentralHello cert bundle: `central/sensor/service/service_impl.go` — Central proactively issues certs and includes them in the CentralHello handshake message. Used by the CRS registration flow; typically ignored by Sensor during normal reconnects.
- Legacy manual cert download (UI/API): `central/certgen/` — generates YAML files for users to `kubectl apply`
- CA rotation logic: `operator/internal/central/carotation/rotation.go`
- Operator TLS reconciliation: `operator/internal/central/extensions/reconcile_tls.go`
- Sensor cert init (copy at startup, watch `certs-new` for updates): `sensor/kubernetes/certinit/init_tls_certs.go`
- Sensor cert refresh (TLS challenge + CA bundle): `sensor/kubernetes/certrefresh/`
- Postgres DB cert reload (Central DB, Scanner V4 DB): `image/postgres/scripts/cert-watcher.sh`, `image/templates/helm/shared/templates/_cert-watcher.tpl`

### Central has three independent cert-handling paths

1. **TLS manager** (`TLSConfigHolder`) — incoming connections. Composes default cert (watched) + internal cert (watched) + OpenShift cert (watched, if configured) + sync.Once trust roots.
   - **OpenShift service-serving TLS** (`central/tlsconfig/openshift_tls.go`) — on OpenShift, watches certs issued for the `central-ocp` Service. Loaded into the TLS manager and presented via SNI on the same `:8443` listener alongside the StackRox internal cert.
2. **Outbound client connections** (`clientconn.TLSConfig`) — reads leaf cert from disk on each TLS handshake, trust pool from `mtls.CACert()`.
3. **TLS challenge endpoint** (`central/metadata/service`) — reads primary leaf via certwatch, issues secondary leaf with short validity and auto-renewal, reads CA via `mtls.CACert()`.

## Certificate caching

- CA certificates are cached at start-up and not reloaded (`CACert()`, `SecondaryCACert()`, `CAForSigning()`, etc.) in `pkg/mtls/crypto.go`: `sync.Once`. The Operator restarts all pods on CA change. Possible improvement: hot reloading CAs would reduce downtime due to pods restarts (this happens only every 1-3 years though).

## Certificate management — who manages what

- **Central side**: the Operator manages all TLS secrets, creates the CA, issues leaf certs, renews them at half validity, and handles CA rotation.
- **Secured Cluster side**: Sensor requests new certs from Central via the cert refresh API and writes them to local Kubernetes secrets. During CA rotation, Sensor and the Operator work together: Sensor writes both CAs (learned from Central) into a CA bundle ConfigMap (`tls-ca-bundle`), and the Operator watches it to update the `caBundle` field on the admission controller's `ValidatingWebhookConfiguration`. This is why full CA rotation requires the Operator also on the Secured Cluster side.

## CA rotation

- Operator-only feature (since 4.9). Orchestrated by the Operator on the Central side through phases: AddSecondary (year 3), PromoteSecondary (year 4), DeleteSecondary (year 5).
- Propagated to Secured Clusters via the cert refresh API and the TLSChallenge endpoint.
- The original design was for only Sensor and Central (the deployment) to be dual-CA aware. In practice, various corner cases led to most deployments gaining dual-CA awareness.
- Current state: Operator, all Central CR components (except legacy Scanner V2), Sensor, and admission controller on the SecuredCluster CR side are dual-CA aware.
- The Operator restarts all Central and SecuredCluster CR pods when a CA change occurs.

### Which CA signs what

- Central services: always signed by primary CA.
- Secured Cluster services: signed by newer CA if Sensor supports rotation; by Sensor's trusted CA (via fingerprint) for Helm clusters; by primary CA otherwise.

### Legacy Helm-managed Secured Clusters (partial CA rotation support)

- Can connect to a rotated Central (Sensor discovers new CA via TLSChallenge).
- However cannot update CA, because they:
  - cannot restart pods on CA change (no Operator)
  - cannot update ValidatingWebhookConfiguration caBundle (no Operator to watch the CA bundle ConfigMap).

## TLS challenge endpoint (/v1/tls-challenge)

- Unauthenticated endpoint, called by Sensor before establishing its main mTLS connection to Central, to bootstrap TLS trust. Sensor sends challenge token, Central returns signed TrustInfo.
- Original purpose: allows Sensor to discover Central's internal CA cert when Sensor cannot validate the certificate that Central presents, e.g when Central uses a custom default certificate. (This endpoint does not help when Central cannot receive Sensor's client certificate.)
- Extended in 4.9 for CA rotation: response includes both primary and secondary cert chains. This is how Sensor (on a different cluster) discovers the new CA — it verifies the response using the CA it already trusts, then adds the other CA to its trust pool.
- Response includes: primary cert chain, secondary cert chain (if present), additional CAs, default TLS leaf cert.
- Signed with both primary and secondary leaf certs. Sensor verifies one signature and trusts all certs in the response (trust delegation).
- Secondary leaf cert: issued in memory from secondary CA with ~3-hour validity, auto-renewed before expiry.
