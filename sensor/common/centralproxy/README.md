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
      "permissionSetID": "b3070020-ecc3-4f34-a3f6-ad28ffc9b80e",
      "httpRequestScopes": {"/metrics": ["GET"]}
    },
    {
      "authType": "kubernetesBearer",
      "subject": "*",
      "permissionSetID": "b3070020-ecc3-4f34-a3f6-ad28ffc9b80b",
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
    }
  ]
}
```

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
outage does not invalidate the last successful watch snapshot. Method/path
scopes are validated locally and must also be included in the token request to
Central. `httpRequestScopes` is a map from canonical paths to nonempty method
lists; repeated JSON path keys are rejected rather than silently overwritten.
An entry must have at least one path. Unsupported versions, unknown/duplicate
JSON fields, and malformed entries reject the entire policy. Policies using the
earlier list form must be changed to the map form before upgrading Sensor.

Sensor authenticates bearer clients via Kubernetes TokenReview and keeps the
existing namespace SubjectAccessReview. Each policy entry's permission set and
HTTP scopes are sent to Central; Central checks that the permission set has no
write access and requires at least one HTTP method before signing the requested
scopes. A request naming only the built-in console or metrics permission set can
omit scopes; Central then signs that set's built-in HTTP profile. Generic sets
must specify HTTP scopes. Older Sensors that send only resource permissions do
not need HTTP scopes. Central enforces signed scopes on HTTP requests. The signed
`requester` identity is for audit only: a stolen bearer token is reusable until
expiry and is **not** proof-of-possession bound to the original client. Sensor
replaces the incoming Authorization header and never sends a Kubernetes token
to Central.
Sensor forwards custom permission sets or HTTP scopes outside its built-in
console/metrics defaults only when Central advertises
`CentralProxyCustomReadOnlyTokens`. Those defaults remain for older Centrals. This
implementation supports OpenShift clusters, whose Sensor proxy uses an OpenShift
service-CA certificate. Cluster scopes do not confine permissions for globally
scoped Central resources. A read-only permission set
also does not prove that every HTTP method/route is side-effect-free; review
each policy entry's route scope. Limit writes to this
ConfigMap and referenced CA ConfigMaps to trusted administrators and Sensor's
service account. Enable `sensor.centralProxy.enabled` in Helm or
`spec.sensor.centralProxyEnabled` in the Operator to serve custom-only clients
without either built-in integration; both default off. Additional clients need
their own ingress and client setup; this option does not create a ServiceMonitor.
