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
`spec.sensor.centralProxyEnabled: true` in the SecuredCluster resource.
Add a NetworkPolicy allowing your clients to reach Sensor on TCP 9444; the
default proxy policy grants ingress only to enabled OpenShift integrations.
Configure clients and any referenced client CA ConfigMaps separately from
this serving certificate.

The ConfigMap `sensor-central-proxy-clients` in Sensor's namespace contains a
versioned `clients.json` policy. Sensor creates defaults only when the ConfigMap
is absent at startup. An invalid existing policy prevents startup. Subsequent
invalid updates or deletion deny requests once observed by the watch; Sensor
does not recreate a deleted policy until restart. An empty `clients` array
disables every client.

The default policy is:

```json
{
  "version": 1,
  "clients": [
    {
      "authType": "kubernetesBearer",
      "subject": "*",
      "permissionSetId": "ffffffff-ffff-fff4-f5ff-fffffffffff2"
    }
  ]
}
```

Each client directly selects exactly one
nonempty `permissionSetId` or `permissionSetName`. Names are portable but must
track permission-set renames; IDs bind to a specific set. Unknown or duplicate
JSON fields, duplicate client identities, malformed selectors/CA references,
unsupported versions, and more than 64 clients reject the entire policy.

Bearer subjects are exact TokenReview usernames or `*`. Only a console selector
may use the wildcard; exact entries override it regardless of order. Certificate
subjects are exact common names and require `caRef`. A presented certificate
prevents bearer fallback. Each request verifies the certificate against **only
its matched entry's current CA roots**, including on reused TLS connections.
Missing, deleted, empty, or malformed CA data denies that client's requests.
Temporary watch failures retain the last observed snapshot. Policy and CA
authorization uses watch snapshots without per-request ConfigMap API reads;
bearer TokenReview and SubjectAccessReview retain their existing caching.

After authentication and client selection:

- **Console-selected clients** retain GET, POST, OPTIONS, and HEAD and Central's
  advertised console path allowlist. The console feature flag must be enabled.
  The selector is the fixed Console Proxy ID or the name
  `OpenShift Secured Cluster Console Proxy`. Console bearer bootstrap may omit
  the namespace header, producing an empty scope.
- **Custom clients** use Central RBAC/SAC without Sensor's console method/path
  restrictions. Exact custom bearer clients require a namespace header and
  successful namespace SubjectAccessReview; `*` requires cluster-wide review.
  Certificate clients request full access to Sensor's cluster.

New mTLS and custom bearer clients require
`CentralProxyPermissionSetTokens`. Only console bearer clients may use the legacy
Deployment/Image token flow on older Central. Issuance errors never trigger a
legacy fallback.

Central resolves the single permission set, requires read-only and
cluster/namespace constraints and matching grants, and limits requested scopes
to the authenticated Sensor's cluster. Sensor replaces the incoming Authorization
header and removes the namespace header before forwarding. Tokens are cached by
identity, selector, scope, cluster, and legacy mode for 55 minutes, with a one-hour
lifetime. Permission reductions can take up to one hour to affect issued tokens.
The signed requester is an audit attribute, not proof of possession. Tokens carry
resource permissions and cluster/namespace scopes, not HTTP restrictions; local
route checks do not constrain direct replay of a stolen token. A read-only
permission set does not establish that every HTTP route is side-effect-free.

Enable `sensor.centralProxy.enabled` in Helm or `spec.sensor.centralProxyEnabled`
in the Operator for custom-only use; these set `ROX_CENTRAL_PROXY_ENABLED` and
default off. The proxy also starts for the built-in console integration, so
those installations can add clients without this flag. Additional clients need
their own ingress and client setup. Policy and referenced CA ConfigMaps should be writable
only by trusted administrators and Sensor's service account.
