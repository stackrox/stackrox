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
        "permissionSetName": "OpenShift Prometheus",
        "httpRequestScopes": {"/metrics": ["GET"]}
      }]
    },
    {
      "authType": "kubernetesBearer",
      "subject": "*",
      "access": [{
        "permissionSetId": "ffffffff-ffff-fff4-f5ff-fffffffffff2",
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
  "access": [{
    "name": "internal role",
    "clusters": {"cluster-id": ["*"]},
    "http": {"/metrics": ["GET"]}
  }]
}
```

The issued token is a signed JWT string (`header.payload.signature`); its
payload also contains time, issuer, and token-ID claims omitted above. The
scoped audience identifies the internal token source with an `-http-scoped`
suffix, which Centrals without HTTP-scope enforcement reject. Each
`access` role carries its own optional `http` map, so multiple policy grants become
separate roles. Each role carries its own `clusters` claim, including roles without
HTTP restrictions. If all grants omit HTTP scopes, the token uses the ordinary
internal-token audience `https://stackrox.io/jwt-sources#internal-rox-tokens`.
`access[].name` is the generic generated role name,
not the policy's permission-set selector. Central resolves the permission set to
resource `permissions` in the role: for example, a Deployment read grant adds
`"permissions": {"read": ["Deployment"]}` alongside `http`. The seeded metrics
set has no resource permissions by default, so `permissions` is absent above.

Bearer `subject` is an exact TokenReview username or `*` for any authenticated
user. The wildcard is reserved for the console permission set; custom bearer
clients must use an exact username. Custom bearer clients require a namespace
scope that passes Sensor's existing SubjectAccessReview. mTLS clients request
full access to Sensor's cluster; exact bearer clients use their authorized
namespace. Central validates any requested cluster against the authenticated
Sensor identity. Only a client with a single console grant retains the existing
optional namespace scope; a console grant combined with other grants is a custom
client regardless of grant order.
Exact entries override the wildcard; duplicate identities are invalid.
mTLS `subject` is an exact certificate common name. A presented certificate
prevents fallback to bearer authentication. The policy verifies that certificate
against **only its entry's watched CA reference**; it does not rely on TLS
`VerifiedChains` from a different CA bundle.
Missing or invalid watched CA data fails closed; a temporary Kubernetes API
outage does not invalidate the last successful watch snapshot. Each client's
nonempty `access` list groups paths and methods by either `permissionSetName`
or `permissionSetId` (exactly one per entry), without repeating credentials.
For example, an exact bearer client can name a
`Deployment Reader` set for `GET /v1/deployments` and an `Image Reader` set
for `GET /v1/images` in two `access` entries. Optional `httpRequestScopes` maps canonical
paths to method lists; paths ending in `/` include descendants. An empty method
list (`[]` or `null`) denies that path, even if a broader prefix allows it.
Maps containing only denials are valid and grant no HTTP access for that role. Omitting
the map or setting it to `null` imposes no HTTP restriction on that grant; an
explicit empty map is invalid. For scoped grants, the most specific matching path
determines access, even if its method is denied. Unrestricted grants still apply
to every route. Distinct permission-set grants may allow the same method and path.
Duplicate permission-set selectors and repeated JSON keys are invalid. Each client
may have at most 32 grants and at most 32 scoped paths and methods in total.
Unsupported versions, unknown/duplicate JSON fields, and malformed entries
reject the entire policy. IDs bind to a specific Central installation; names
are portable but must be updated if the set is renamed. The default Console
Proxy client uses its fixed ID; clients that select by name receive the
permissions of the set Central resolves under that name.

Sensor authenticates bearer clients via Kubernetes TokenReview and keeps the
existing namespace SubjectAccessReview. Sensor checks the selected route locally
and sends all of the client's access grants to Central in the token request's
`permission_sets` field. Central resolves each grant separately and signs separate
roles. Unrestricted roles and scoped roles matching the request's most specific
path and method are available to Central's authorization checks. Direct gRPC
requests can use only unrestricted roles. Central requires each referenced permission set to have
both the read-access and cluster/namespace-scope constraints and to have
grants that satisfy those constraints. Newly seeded console and Prometheus
permission sets have both constraints; pre-existing administrator-managed sets
must be configured with both before Sensor tokens can use them. If the Console
Proxy name is already occupied by a different set, Central logs the seeding
error and the default client cannot resolve its fixed ID. The default mTLS
metrics client requests only `GET /metrics`; Central validates requested HTTP
scopes generically rather than restricting a permission set by name. The default console and metrics
grants retain explicit routes. Older Sensors send only resource permissions;
Central accepts those requests without HTTP scopes. When `permission_sets` is
specified, new Centrals ignore the legacy resource-permissions fallback.
Centrals that issue HTTP-scoped tokens enforce the signed scopes on HTTP requests.
When a JWT also carries a token-wide `http` map, it restricts every role.
Roles without their own `http` map inherit that restriction; roles with one must
satisfy both levels. Most-specific-path matching is evaluated independently at
each level, so a more specific token-wide allow cannot override a role's denial,
or vice versa. An absent map imposes no restriction at that level; an explicitly
empty map is invalid.
Sensor requests the same cluster/namespace scope for all of a client's grants.
The signed `requester` identity is for audit only: a stolen
bearer token is reusable until expiry and is **not** proof-of-possession bound
to the original client. Sensor replaces the incoming Authorization header and
never sends a Kubernetes token to Central.

New metrics and custom clients require `CentralProxyPerAccessTokens`, including
clients with unrestricted grants. Centrals predating this capability cannot
resolve these grants. The built-in console
profile can still use the pre-existing console token flow with those Centrals,
which cannot enforce custom HTTP scopes. This fallback requires a single console
grant with explicit scopes contained in the default console routes and methods.
This implementation supports OpenShift
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
