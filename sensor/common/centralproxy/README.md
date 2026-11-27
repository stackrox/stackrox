# Sensor Central proxy client policy

Sensor serves `/proxy/central/` on Kubernetes and OpenShift.
OpenShift supplies its service-CA serving certificate. On native Kubernetes,
create a `kubernetes.io/tls` Secret named `sensor-proxy-tls` in Sensor's namespace
before enabling the proxy. Its `tls.crt` and `tls.key` must serve
`sensor-proxy.<namespace>.svc` (or the DNS name used by clients), and clients
must trust the issuing CA. For example:

```sh
kubectl -n stackrox create secret tls sensor-proxy-tls --cert=server.crt --key=server.key
```

Enable `sensor.centralProxy.enabled=true` in Helm or
`spec.sensor.centralProxyEnabled: true` in the SecuredCluster resource on
native Kubernetes, where the serving certificate is provisioned manually.
On OpenShift the proxy starts without any flag. Add a NetworkPolicy allowing
your clients to reach Sensor on TCP 9444; the default proxy policy grants
ingress only to enabled OpenShift integrations.
Configure clients and any referenced client CA ConfigMaps separately from
this serving certificate.

The ConfigMap `sensor-central-proxy-clients` in Sensor's namespace contains a
versioned `clients.json` policy. Sensor creates defaults only when the ConfigMap
is absent at startup. An invalid existing policy prevents startup. Subsequent
invalid updates or deletion deny requests once observed by the watch; Sensor
does not recreate a deleted policy until restart. An empty `clients` array
disables every client.

The built-in clients are OpenShift integrations. On OpenShift the default
policy is:

```json
{
  "version": 1,
  "clients": [
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

On other clusters the default policy has no clients: Sensor admits no client
until one is configured.

This policy shape uses a nonempty `access`
array on each client. Each grant selects exactly one nonempty `permissionSetId`
or `permissionSetName` and may specify `httpRequestScopes`. Names are portable but must
track permission-set renames; IDs bind to a specific set. Unknown or duplicate
JSON fields, duplicate client identities, malformed selectors/CA references,
unsupported versions, and more than 64 clients reject the entire policy.
Each client may have at most 32 grants and 32 HTTP paths and methods in total.
Duplicate selectors are invalid; Central also rejects aliases that resolve
multiple grants to the same permission set.

Bearer subjects are exact TokenReview usernames or `*`. Only a single console
grant may use the wildcard; exact entries override it regardless of order. Certificate
subjects are exact common names and require `caRef`. A presented certificate
prevents bearer fallback. Each request verifies the certificate against **only
its matched entry's current CA roots**, including on reused TLS connections.
Missing, deleted, empty, or malformed CA data denies that client's requests.
Temporary watch failures retain the last observed snapshot. Policy and CA
authorization uses watch snapshots without per-request ConfigMap API reads;
bearer TokenReview and SubjectAccessReview retain their existing caching.

The explicit HTTP maps define local route checks. The default console grant
permits the routes and methods shown above. Administrators can configure
different maps for any permission set. The console feature flag applies to
clients with exactly one console grant, selected by the fixed ID or the name
`OpenShift Secured Cluster Console Proxy`. Only these console bearer clients
may omit the namespace header for bootstrap, producing an empty scope. A
console grant combined with another permission set is a custom client
regardless of grant order.

Exact custom bearer clients require a namespace header and successful namespace
SubjectAccessReview; `*` requires cluster-wide review. Certificate clients request
full access to Sensor's cluster. Every grant in a client receives the same
cluster/namespace scope.

`httpRequestScopes` maps canonical paths to uppercase HTTP methods. A trailing
slash matches descendants; other paths match exactly. An empty or null method
list denies that path. The most specific matching path across the client's
scoped grants determines which methods can apply; a broader prefix cannot
override a more specific denial. Multiple grants may permit the same path and
method. Omitting a grant's HTTP map, or setting it to null, imposes no HTTP
restriction on that grant. An explicitly empty map is invalid. Unrestricted
grants remain available on every route, including routes denied by scoped grants.
Canonical-path checks reject ambiguous/encoded paths for scoped grants.

New mTLS and custom bearer clients require
`CentralProxyPermissionSetTokens`. Only console bearer clients may use the legacy
Deployment/Image token flow on older Central. Issuance errors never trigger a
legacy fallback.

Central resolves each permission set separately, requires read-only and
cluster/namespace constraints and matching grants, and limits requested scopes
to the authenticated Sensor's cluster. Sensor replaces the incoming Authorization
header and removes the namespace header before forwarding. Each token request
carries only the grants whose HTTP scope allows the current request, so grants
scoped to other routes never widen a forwarded request. Tokens are cached by
identity, the allowed grants, cluster/namespace scope, cluster, and legacy mode
for 55 minutes, with a one-hour lifetime. Permission reductions can take up to one hour
to affect issued tokens. The signed requester is an audit attribute, not proof
of possession. Tokens carry resource permissions and cluster/namespace scopes,
not HTTP restrictions; local route checks do not constrain direct replay of a
stolen token. A read-only permission set does not establish that every HTTP
route is side-effect-free.

Enable `sensor.centralProxy.enabled` in Helm or `spec.sensor.centralProxyEnabled`
in the Operator on native Kubernetes, where these set `ROX_CENTRAL_PROXY_ENABLED`
and default off alongside the manually provisioned serving-certificate Secret.
On OpenShift the proxy starts without any flag. Additional clients need
their own ingress and client setup. Policy and referenced CA ConfigMaps should be writable
only by trusted administrators and Sensor's service account.
