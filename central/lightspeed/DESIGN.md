# OpenShift Lightspeed Integration — Prototype Design

## Problem

ACS has no integration with OpenShift Lightspeed, an AI assistant for OpenShift
clusters. We want to leverage Lightspeed to provide AI-powered risk summaries for
deployments monitored by ACS.

## Goal

Build a prototype that:
1. Lets users configure a Lightspeed endpoint per secured cluster
2. Periodically validates connectivity and authorization from Sensor
3. Provides an API endpoint that sends deployment risk data to Lightspeed and
   returns an AI-generated summary

## Architecture

```
User -> Central API (/v1/deploymentswithrisk/{id}/lightspeed-summary)
          |
        Central (broker: correlate request/response by UUID)
          | MsgToSensor (gRPC stream)
        Sensor (querier component)
          | HTTP POST with SA token
        Lightspeed (/v1/query on the configured host)
          |
        Response flows back: Sensor -> Central -> User
```

Configuration flow (separate, periodic):
```
User -> PUT /v1/clusters/{id}/lightspeed-config {host: "https://..."}
          |
        Central stores host, sends MsgToSensor{LightspeedConfig}
          |
        Sensor (updater component, every 30s):
          GET {host}/readiness  (is the service up?)
          POST {host}/authorized (does the SA token work?)
          Sends MsgFromSensor{LightspeedInfo} back to Central
          |
        Central pipeline stores status in-memory
```

## Why route through Sensor?

Central does not have direct network access to services running inside secured
clusters. The Sensor gRPC stream is the established communication channel.
Sensor already has an auto-mounted service account token and can reach
cluster-local services. This follows the existing `DeploymentEnhancement`
pattern where Central sends requests to Sensor for cluster-local operations.

## Why user-configured host instead of auto-detection?

Lightspeed deployment names and namespaces can be customized by the operator
configuration (`OLSConfig` CR). Rather than guessing deployment names, we let
the user provide the service URL. Sensor validates it using Lightspeed's
`/readiness` and `/authorized` endpoints.

## Authentication

Sensor's pod has an auto-mounted service account token at
`/var/run/secrets/kubernetes.io/serviceaccount/token`. Lightspeed uses
`kube-rbac-proxy` fronting, which validates tokens via Kubernetes
`TokenReview`. The Sensor SA needs the `lightspeed-operator-query-access`
ClusterRole binding to be authorized. This binding is included in the
secured-cluster Helm chart at
`templates/sensor-lightspeed-rbac.yaml`, gated on
`lightspeed.enabled: true` in the Helm values:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: stackrox-lightspeed-access
subjects:
- kind: ServiceAccount
  name: sensor
  namespace: stackrox
roleRef:
  kind: ClusterRole
  name: lightspeed-operator-query-access
  apiGroup: rbac.authorization.k8s.io
```

## Patterns followed

1. **Compliance Operator detection** (`sensor/kubernetes/complianceoperator/updater.go`)
   for the periodic health check and Central storage flow.

2. **DeploymentEnhancement broker** (`central/sensor/enhancement/broker.go` +
   `sensor/common/deploymentenhancer/component.go`) for the Central-to-Sensor
   request/response proxy with UUID correlation.

## Prototype scope

- In-memory storage only (no database tables, no migrations)
- No feature flags or capability negotiation (assumes current Sensor)
- Hardcoded prompt for risk summarization
- `InsecureSkipVerify` for cluster-internal TLS (prototype only)
- Config lost on Central restart (Sensor re-sends status every 30s, but host
  config must be re-set)

## Security follow-ups

These are incremental hardening tasks that don't require architectural changes:

1. **Host validation**: `ConfigureLightspeed` should reject non-HTTPS URLs and
   restrict accepted hosts to in-cluster service patterns (`.svc` /
   `.svc.cluster.local` suffix). This prevents pointing Sensor at an
   attacker-controlled endpoint that could capture the SA token.

2. **TLS certificate verification**: Remove `InsecureSkipVerify` from the Sensor
   HTTP client. Load the cluster CA from
   `/var/run/secrets/kubernetes.io/serviceaccount/ca.crt` into
   `tls.Config.RootCAs` to verify Lightspeed's certificate.

3. **Stale status race**: `Store.UpdateInfo` should ignore status updates where
   `info.Host` doesn't match the currently configured host for that cluster.
   After a host reconfiguration from A to B, an in-flight status report from A
   should not overwrite the entry for B.

4. **Payload minimization**: `GetDeploymentRiskSummary` serializes the full
   deployment and risk proto as context. Redact sensitive fields (env vars,
   service accounts, image pull secrets) before sending to Lightspeed.

5. **Context cancellation in broker**: `SendAndWaitForSummary` should watch
   `ctx.Done()` in the select, not just `sig.arrived` and `time.After`. A
   canceled gRPC request currently blocks for up to 60s.

6. **Nil status check**: `GetDeploymentRiskSummary` should treat `info == nil`
   the same as not-ready, failing fast instead of proceeding to the broker.

## API Endpoints

### Configure Lightspeed
```
PUT /v1/clusters/{cluster_id}/lightspeed-config
Body: {"host": "https://lightspeed-app-server.openshift-lightspeed.svc:8443"}
```

### Get Lightspeed status
```
GET /v1/clusters/{id}/lightspeed-config
Response: {"host": "...", "is_ready": true, "has_query_access": true}
```

### Get deployment risk summary
```
GET /v1/deploymentswithrisk/{id}/lightspeed-summary
Response: {"summary": "This deployment has a high risk score of 4.2 due to..."}
```

## Wire format (proto messages)

- `central.LightspeedConfig` — Central to Sensor: configure the host URL
- `central.LightspeedInfo` — Sensor to Central: readiness + auth status
- `central.LightspeedQueryRequest` — Central to Sensor: query with prompt + context JSON
- `central.LightspeedQueryResponse` — Sensor to Central: summary or error

All messages flow through the existing `MsgToSensor` / `MsgFromSensor` oneof
on the bidirectional gRPC stream.
