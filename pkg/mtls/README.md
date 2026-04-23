# TLS certificates in StackRox

## Architecture

- StackRox uses mTLS for most inter-service communication. Service type is extracted
  from the client TLS certificate for RPC authorization.
- Exceptions (TLS but not mTLS): Postgres connections, config-controller.
- Internal CA: self-signed, 5-year validity, stored in `central-tls` secret.
  Only Central and the Operator have the CA private key.
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

## Key code locations

- Service identity extraction from client cert CN: `pkg/mtls/cn.go`, `pkg/grpc/authn/service/extractor.go`
- CA loading (sync.Once, process-lifetime cache): `pkg/mtls/crypto.go`
- Leaf cert loading (no cache, reads disk each call): `mtls.LeafCertificateFromFile()` in `pkg/mtls/crypto.go`
- Cert file watcher: `pkg/mtls/certwatch/certwatch.go` — polls directory every 5s, debounces, calls update callback
- TLS config composition: `pkg/mtls/certwatch/tls_config_holder.go` — `atomic.Pointer[tls.Config]`, rotates session ticket keys on every update to invalidate cached TLS sessions
- Trust pool builders: `pkg/mtls/verifier/verify.go` — `TrustedCertPool()`, `NonCA.TLSConfig()`
- Central TLS manager: `central/tlsconfig/manager_impl.go` — composes server certs + trust roots for incoming connections
- Central TLS cert loaders: `central/tlsconfig/tlsconfig.go` — `loadInternalCertificateFromFiles()`, `MaybeGetDefaultTLSCertificateFromDirectory()`
- TLS challenge endpoint: `central/metadata/service/service_impl.go`
- Cert issuance for Secured Clusters: `central/securedclustercertgen/certificates.go`
- CA rotation logic: `operator/internal/central/carotation/rotation.go`
- Operator TLS reconciliation: `operator/internal/central/extensions/reconcile_tls.go`
- Sensor cert init (one-time copy at startup): `sensor/kubernetes/certinit/init_tls_certs.go`
- Sensor cert refresh (TLS challenge + CA bundle): `sensor/kubernetes/certrefresh/`

## Caching behavior

### Read once per process (sync.Once in pkg/mtls/crypto.go)

- `CACert()`, `CACertPEM()`, `SecondaryCACert()` — CA cert public bytes
- `CAForSigning()`, `SecondaryCAForSigning()` — CA signing objects
- CA key bytes
- These NEVER refresh without a pod restart.

### Hot-reloaded via certwatch

- Default/ingress TLS cert — `certwatch.WatchCertDir(DefaultCertPath, ...)` in `central/tlsconfig/manager_impl.go`
- Secure metrics TLS cert — `certwatch.WatchCertDir` in `pkg/metrics/tls.go`

### Loaded once at startup, never refreshed

- Central internal service leaf cert — loaded in `getInternalCertificates()` at TLS manager construction, stored in `internalCerts`
- Central primary leaf for TLS challenge — `sync.Once` in `central/metadata/service/service_impl.go`
- Central secondary leaf for TLS challenge — `sync.Once`, issued in memory from secondary CA
- Scanner V4 (indexer/matcher): server cert via `verifier.NonCA{}`, client cert via `clientconn.TLSConfig`
- Admission controller: webhook server cert via `verifier.NonCA{}`
- Sensor: client cert loaded in `centralclient.NewClient`, proxy cert in `StartProxyServer`
- Config controller: CA pool via `verifier.SystemCertPool()`

## Central has three independent cert-handling paths

1. **TLS manager** (`TLSConfigHolder`) — incoming connections. Composes default cert (watched) + internal cert (loaded once) + sync.Once trust roots.
2. **Outbound client connections** (`clientconn.TLSConfig`) — reads leaf from disk per connection, trust pool from `mtls.CACert()`.
3. **TLS challenge endpoint** (`central/metadata/service`) — reads primary leaf via `sync.Once`, issues secondary leaf via `sync.Once`, reads CA via `mtls.CACert()`.

## CA rotation

- Operator-only feature (since 4.9). Phases: AddSecondary (year 3), PromoteSecondary (year 4), DeleteSecondary (year 5).
- Dual-CA awareness is limited to: Operator, Central, Sensor, admission controller ValidatingWebhookConfiguration.
- All other intra-cluster components only know a single CA. The Operator restarts them on CA change via a pod template annotation hash that includes the CA PEM (`confighash.ComputeCAHash`).
- This was deliberate: the CA rotation problem is primarily Central↔Sensor (cross-cluster). Intra-cluster services can be restarted together.

### Which CA signs what

- Central services: always signed by primary CA.
- Secured Cluster services: signed by newer CA if Sensor supports rotation; by Sensor's trusted CA (via fingerprint) for Helm clusters; by primary CA otherwise.

### Helm-managed Secured Clusters (partial CA rotation support)

- Can connect to a rotated Central (Sensor discovers new CA via TLSChallenge).
- Cannot restart pods on CA change (no Operator).
- Cannot update ValidatingWebhookConfiguration caBundle (main blocker for full Helm CA rotation).

## TLS challenge endpoint (/v1/tls-challenge)

- Unauthenticated endpoint. Sensor sends challenge token, Central returns signed TrustInfo.
- Response includes: primary cert chain, secondary cert chain (if present), additional CAs, default TLS leaf cert.
- Signed with both primary and secondary leaf certs. Sensor verifies one signature and trusts all certs in the response (trust delegation).
- Secondary leaf cert: issued in memory from secondary CA with 1-year validity via sync.Once, never renewed.

## Sensor certinit — blocks hot reload

`sensor/kubernetes/certinit/init_tls_certs.go` copies certs from source mounts
(`certs-new` or `certs-legacy`) to `CertsPrefix` at startup. This is a ONE-TIME
COPY to an emptyDir. After startup, Secret updates to the source volume do NOT
propagate to the files the process reads. This blocks any hot-reload for Sensor
until certinit is made continuous or removed.
