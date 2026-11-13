# Sensor Central proxy client policy

The ConfigMap `sensor-central-proxy-clients` in the Sensor pod namespace contains
`clients.json`. `NewClientPolicy(kubernetesClient).EnsureDefaults(ctx)` creates it
only when it is absent at startup. An invalid existing ConfigMap prevents the
proxy from starting; deleting it after startup denies subsequent requests once
the watch observes the deletion, until Sensor restarts (which recreates the
defaults).
Changes to the ConfigMap take effect when its watch delivers the update, without
restarting Sensor. Requests use locally cached policy and CA data, not Kubernetes
API reads.
An empty `clients` array disables all clients without deleting the ConfigMap.

The default document has this shape (the actual CN and CA reference come from
`env.SecureMetricsClientCertCN` and `env.SecureMetricsClientCA*`):

```json
{
  "version": 1,
  "clients": [
    {
      "authType": "mtls",
      "subject": "system:serviceaccount:openshift-monitoring:prometheus-k8s",
      "caRef": {
        "namespace": "kube-system",
        "configMap": "extension-apiserver-authentication",
        "key": "client-ca-file"
      },
      "access": [{
        "permissionSetName": "OpenShift Prometheus Metrics Reader",
        "httpRequestScopes": {"/metrics": ["GET"]}
      }]
    },
    {
      "authType": "kubernetesBearer",
      "subject": "*",
      "access": [{
        "permissionSetName": "OpenShift Secured Cluster Console Proxy",
        "httpRequestScopes": {
          "/api/graphql": ["POST"],
          "/static/ocp-plugin/": ["GET"],
          "/v1/config/public": ["GET"],
          "/v1/deployments": ["GET"],
          "/v1/featureflags": ["GET"],
          "/v1/metadata": ["GET"],
          "/v1/mypermissions": ["GET"],
          "/v1/telemetry/": ["GET"]
        }
      }]
    }
  ]
}
```

For the mTLS `/metrics` client above, the **decoded JWT payload** issued by a
Central supporting per-access tokens looks like this (selected claims shown;
the cluster ID is illustrative):

```json
{
  "aud": "https://stackrox.io/jwt-sources#internal-rox-tokens-http-scoped",
  "req": "mtls:system:serviceaccount:openshift-monitoring:prometheus-k8s",
  "clusters": {"cluster-id": ["*"]},
  "access": [{
    "name": "internal role",
    "http": {"/metrics": ["GET"]}
  }]
}
```

The issued token is a signed JWT string (`header.payload.signature`); its
payload also contains time, issuer, and token-ID claims omitted above. The
scoped audience identifies the internal token source with an `-http-scoped`
suffix, which Centrals without HTTP-scope enforcement reject. Each
`access` role carries its own `http` map, so multiple policy grants become
separately scoped roles. `access[].name` is the generic generated role name,
not the policy's `permissionSetName`. Central resolves the permission set to
resource `permissions` in the role: for example, a Deployment read grant adds
`"permissions": {"read": ["Deployment"]}` alongside `http`. The seeded metrics
set has no resource permissions by default, so `permissions` is absent above.

Bearer `subject` is an exact TokenReview username or `*` for any authenticated
user. The wildcard is reserved for the console permission set; custom bearer
clients must use an exact username. Custom bearer clients require a namespace
scope that passes Sensor's existing SubjectAccessReview. mTLS clients (with any
permission set) and bearer clients using the metrics set request full access to
Sensor's cluster, which Central validates against the authenticated Sensor
identity. The console set retains its existing optional namespace scope.
Exact entries override the wildcard; duplicate identities are invalid.
mTLS `subject` is an exact certificate common name. A presented certificate
prevents fallback to bearer authentication. The policy verifies that certificate
against **only its entry's watched CA reference**; it does not rely on TLS
`VerifiedChains` from a different CA bundle.
Missing or invalid watched CA data fails closed; a temporary Kubernetes API
outage does not invalidate the last successful watch snapshot. Each client's
nonempty `access` list groups paths and methods by `permissionSetName`, without
repeating credentials. For example, an exact bearer client can name a
`Deployment Reader` set for `GET /v1/deployments` and an `Image Reader` set
for `GET /v1/images` in two `access` entries. `httpRequestScopes` maps canonical
paths to nonempty method lists; paths ending in `/` include descendants. The
most specific matching path determines access, even if its method is denied.
Duplicate permission-set names or overlapping path/method pairs within a client
are invalid, as are repeated JSON keys. An entry must have at least one path.
Unsupported versions, unknown/duplicate JSON fields, and malformed entries
reject the entire policy. Names are resolved by Central, so update a renamed
permission set in this ConfigMap or the client cannot obtain a token.

Sensor authenticates bearer clients via Kubernetes TokenReview and keeps the
existing namespace SubjectAccessReview. Sensor checks the selected route locally
and sends all of the client's access grants to Central, which signs separate
HTTP-scoped roles with a shared cluster or namespace scope. Only the roles
matching the request's most specific path and method are available to Central's
authorization checks. Central requires each referenced permission set to have
both the read-access and cluster/namespace-scope constraints. Newly seeded
console and Prometheus permission sets have both constraints; pre-existing
administrator-managed sets must be configured with both before Sensor tokens can
use them. The built-in metrics token is restricted to `GET /metrics` and cannot
be combined with other permission sets. Custom scrapers need their own permission
set if their grants differ from the built-in Prometheus reader. At least one HTTP
method is required.
A request naming a permission set must include explicit HTTP method/path scopes,
including for the built-in console and metrics sets. Sensors using the
pre-existing console proxy still send only resource permissions; Central accepts
those requests without HTTP scopes. Centrals that issue HTTP-scoped tokens
enforce the signed scopes on HTTP requests. A client's grants must share a
single cluster/namespace scope; metrics grants cannot be combined with other
permission sets. The signed `requester` identity is for audit only: a stolen
bearer token is reusable until expiry and is **not** proof-of-possession bound
to the original client. Sensor replaces the incoming Authorization header and
never sends a Kubernetes token to Central.

New metrics and custom clients require `CentralProxyPerAccessTokens` to issue
HTTP-scoped tokens, including tokens with multiple access grants. Centrals
predating this capability cannot issue these tokens. The built-in console
profile can still use the pre-existing console token flow with those Centrals,
which cannot enforce custom HTTP scopes. This implementation supports OpenShift
clusters, whose Sensor proxy uses an OpenShift service-CA certificate.

Cluster scopes do not confine permissions for globally scoped Central resources.
A read-only permission set also does not prove that every HTTP method/route is
side-effect-free; review each policy entry's route scope. Limit writes to this
ConfigMap and referenced CA ConfigMaps to trusted administrators and Sensor's
service account. Enable `sensor.centralProxy.enabled` in Helm or
`spec.sensor.centralProxyEnabled` in the Operator to serve custom-only clients
without either built-in integration; both default off. Additional clients need
their own ingress and client setup; this option does not create a ServiceMonitor.
The general proxy switch does not enable `/metrics`: that endpoint still requires
the separate secured-cluster metrics opt-in. On installations already running
the proxy for the console or metrics integration, administrators can configure
additional clients without enabling the general proxy switch.
