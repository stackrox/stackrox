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
- Central TLS cert loaders: `central/tlsconfig/tlsconfig.go` — `LoadInternalCertificateFromDirectory()`, `MaybeGetDefaultTLSCertificateFromDirectory()`
- TLS challenge endpoint: `central/metadata/service/service_impl.go`
- Cert issuance for Secured Clusters: `central/securedclustercertgen/certificates.go`
- CentralHello cert bundle: `central/sensor/service/service_impl.go` — Central proactively issues certs and includes them in the CentralHello handshake message. Used by the CRS registration flow; typically ignored by Sensor during normal reconnects.
- Legacy manual cert download (UI/API): `central/certgen/` — generates YAML files for users to `kubectl apply`
- CA rotation logic: `operator/internal/central/carotation/rotation.go`
- Operator TLS reconciliation: `operator/internal/central/extensions/reconcile_tls.go`
- Sensor cert init (one-time copy at startup): `sensor/kubernetes/certinit/init_tls_certs.go`
- Sensor cert refresh (TLS challenge + CA bundle): `sensor/kubernetes/certrefresh/`

### Central has three independent cert-handling paths

1. **TLS manager** (`TLSConfigHolder`) — incoming connections. Composes default cert (watched) + internal cert (watched) + sync.Once trust roots.
2. **Outbound client connections** (`clientconn.TLSConfig`) — reads leaf from disk per connection, trust pool from `mtls.CACert()`.
3. **TLS challenge endpoint** (`central/metadata/service`) — reads primary leaf via certwatch, issues secondary leaf with short validity and auto-renewal, reads CA via `mtls.CACert()`.

## Certificate caching

The following certificates are currently known to be cached at start-up and not reloaded:

- CA material (`CACert()`, `SecondaryCACert()`, `CAForSigning()`, etc.) in `pkg/mtls/crypto.go`: `sync.Once`, never refreshed. Intentional — the Operator restarts all pods on CA change.

- Sensor: all certs are effectively cached because `certinit` copies them to an emptyDir at startup. Client certs are also cached at construction (`centralclient.NewClient`, `StartProxyServer`, scanner client).
- Scanner V4 (indexer/matcher) client certs: cached at dial time via `clientconn.TLSConfig`
- Admission controller client cert for Sensor connection: `clientconn.AuthenticatedGRPCConnection` at startup
- Compliance client cert for Sensor connection: `clientconn.AuthenticatedGRPCConnection` at startup
- Postgres (Central DB, Scanner DB, Scanner V4 DB): need SIGHUP to reload SSL certs

## Certificate management — who manages what

- **Central side**: the Operator manages all TLS secrets, creates the CA,
  issues leaf certs, renews them at half validity, and handles CA rotation.
- **Secured Cluster side**: Sensor requests new certs from Central via the
  cert refresh API and writes them to local Kubernetes secrets. During CA
  rotation, Sensor and the Operator work together: Sensor writes both CAs
  (learned from Central) into a CA bundle ConfigMap (`tls-ca-bundle`), and the
  Operator watches it to update the `caBundle` field on the admission
  controller's `ValidatingWebhookConfiguration`. This is why full CA rotation
  requires the Operator on the Secured Cluster side.

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
- Cannot update ValidatingWebhookConfiguration caBundle (no Operator to watch the CA bundle ConfigMap).

## TLS challenge endpoint (/v1/tls-challenge)

- Unauthenticated endpoint. Sensor sends challenge token, Central returns signed TrustInfo.
- Response includes: primary cert chain, secondary cert chain (if present), additional CAs, default TLS leaf cert.
- Signed with both primary and secondary leaf certs. Sensor verifies one signature and trusts all certs in the response (trust delegation).
- Secondary leaf cert: issued in memory from secondary CA with ~3-hour validity, auto-renewed before expiry.
