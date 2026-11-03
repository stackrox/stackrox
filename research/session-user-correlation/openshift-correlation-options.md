# OpenShift-Native Session Correlation Options

Research note for attributing runtime file activity in OpenShift containers to the authenticated
OpenShift user behind an `oc exec` session. ACS is a read-only consumer of whatever identity
mechanism OpenShift provides. Companion documents: `PRIVILEGED_SESSION_ATTRIBUTION_RESEARCH.md`
(the OpenShift platform design target) and the Oct 2026 lab notes on the forgeable argv marker.

## Summary

No shipped OpenShift mechanism carries the API-server identity of an `oc exec` request to the
node: the CRI `ExecRequest` has no user or session field, kubelet only sees the apiserver's
client identity (as `nodes/proxy`), and CRI-O knows the exec PID but no request context. The
two consumable mechanisms today are (1) API-server audit events — user identity without
runtime PIDs, joined only by time and target — and (2) mutating admission on `pods/exec`,
which plants a request UID in the exec process environment; the Security Profiles Operator
implements this pattern and documents its forgeability honestly. The supported
ClusterLogForwarder audit feed is seconds-class; fail-closed enforcement is possible via
validating webhook plus RBAC on `pods/exec`; adversary-resistant session-to-PID binding needs
the design target's platform change; everything else is best-effort correlation.

## Findings

### 1. Exec/attach audit event schema

**Event fields.** An `oc exec` reaches the Kubernetes API server as a request to the `pods/exec`
subresource and is recorded as an `audit.k8s.io/v1` `Event` with: `level`, `auditID` (unique per
request), `stage`, `requestURI`, `verb`, `user` (`authentication/v1.UserInfo`: username, uid,
groups, extra), `impersonatedUser` (present when impersonation/`--as` is used), `sourceIPs`
(proxied chain; all but the last IP can be arbitrarily set by the client), `userAgent`
(client-provided, explicitly "must not be trusted"), `objectRef` (resource `pods`, subresource
`exec`, namespace, pod name, pod uid), `responseStatus`, `requestObject` (at `Request` level
and higher), `responseObject`, `requestReceivedTimestamp`, `stageTimestamp`, and `annotations`
(included already at `Metadata` level)
([audit Event reference](https://kubernetes.io/docs/reference/config-api/apiserver-audit.v1/);
OCP field table with a full example event:
[audit log docs](https://raw.githubusercontent.com/openshift/openshift-docs/main/modules/nodes-nodes-audit-log-basic.adoc)).

**Stages.** Defined stages are `RequestReceived`, `ResponseStarted` (only for long-running
requests — exec/attach streams qualify), `ResponseComplete`, and `Panic`
([auditing docs](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/)). For an exec, the
same `auditID` therefore yields a `ResponseStarted` event when the stream starts and a
`ResponseComplete` event when it ends — an implicit session-start/end signal, though nothing
ties either to a runtime PID.

**Verb and requestURI.** SPDY-based exec uses HTTP POST (kube verb `create`); WebSocket-based
exec (modern kubectl) uses GET (verb `get`), which is why `get` on `pods/exec` is effectively an
execute permission — a new feature gate re-checks `create` authority on the WebSocket path
([kubernetes issue 133515](https://github.com/kubernetes/kubernetes/issues/133515);
[ExecREST source](https://raw.githubusercontent.com/kubernetes/kubernetes/master/pkg/registry/core/pod/rest/subresources.go)).
`PodExecOptions` (`stdin`, `stdout`, `stderr`, `tty`, `container`, `command`) is converted from
`net/url.Values`, so the command and target container arrive as query parameters and are
recorded verbatim in `requestURI` even at `Metadata` level
([PodExecOptions source](https://raw.githubusercontent.com/kubernetes/api/master/core/v1/types.go);
[kubelet reads them back from the query string](https://raw.githubusercontent.com/kubernetes/kubernetes/master/pkg/kubelet/server/server.go)).

**Annotations.** `authorization.k8s.io/decision` and `authorization.k8s.io/reason` record the
RBAC decision and which binding allowed it
([audit annotations reference](https://kubernetes.io/docs/reference/labels-annotations-taints/audit-annotations/)).
Admission webhooks can add their own annotations to the request's audit event, keys prefixed
with the webhook name (e.g. `execmetadata.spo.io/SPO_EXEC_REQUEST_UID`), via
`AdmissionResponse.auditAnnotations`
([admission API types](https://raw.githubusercontent.com/kubernetes/api/master/admission/v1/types.go)).

**OpenShift additions.** OCP authenticators populate `user.extra` with
`authentication.kubernetes.io/credential-id` (credential fingerprint, e.g. `X509SHA256=...`)
([apiserver user.go](https://raw.githubusercontent.com/kubernetes/apiserver/master/pkg/authentication/user/user.go)).
The OAuth server emits login/logout audit events with
`authentication.openshift.io/username` and `authentication.openshift.io/decision` annotations,
and `oauthaccesstokens`/`oauthauthorizetokens` writes are audited with request bodies (their
sha256-hashed names are not secrets), giving a login/logout trail and a token-to-user mapping
([login-logout enhancement](https://raw.githubusercontent.com/openshift/enhancements/master/enhancements/authentication/login-logout-events.md);
[audit-policy enhancement](https://raw.githubusercontent.com/openshift/enhancements/master/enhancements/kube-apiserver/audit-policy.md)).
There is no session identifier beyond `auditID`, and no PID, container ID, or node in the event
(node is only derivable by resolving the pod separately).

**Audit policy level required on OCP.** OCP configures audit policy through the `APIServer`
cluster resource (`spec.audit.profile`), applied to kube-apiserver, openshift-apiserver and
oauth-apiserver. The default profile `Default` logs metadata for all read and write requests —
exec included — plus token-object bodies; `WriteRequestBodies`/`AllRequestBodies` add request
bodies; `None` disables logging ([OCP audit profile docs](https://raw.githubusercontent.com/openshift/openshift-docs/main/modules/nodes-nodes-audit-config-about.adoc),
[profile configuration](https://raw.githubusercontent.com/openshift/openshift-docs/main/modules/nodes-nodes-audit-policy.adoc)).
Since exec options travel as query parameters, `Default` already captures user, groups,
credential-id, sourceIPs, userAgent, requestURI (with command and container), auditID,
objectRef, and annotations — full metadata needs no profile change. `spec.audit.customRules`
select profiles per *group* only, first match wins; resource-level rules like upstream's
`resources: ["pods/exec"]` are not expressible
([custom rules](https://raw.githubusercontent.com/openshift/openshift-docs/main/modules/nodes-nodes-audit-policy-custom.adoc);
[APIServer type](https://raw.githubusercontent.com/openshift/api/master/config/v1/types_apiserver.go)).
Audit files live on control-plane nodes, kube-apiserver at `/var/log/kube-apiserver/audit.log`
([ComplianceAsCode OCP rule](https://raw.githubusercontent.com/ComplianceAsCode/content/master/applications/openshift/api-server/api_server_audit_log_path/rule.yml)).

### 2. Near-real-time audit consumption

**Webhook backend.** Upstream kube-apiserver ships an audit webhook backend
(`--audit-webhook-config-file`); in the default `batch` mode the maximum wait before a batch is
flushed is 30 seconds (400 events/batch, throttled at 10 batches/s), with a `blocking` mode
that emits synchronously ([auditing docs](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/)).
On OpenShift this is not available through the supported surface: the `APIServer` CR exposes
only `profile` and group-based `customRules` ([type definition](https://raw.githubusercontent.com/openshift/api/master/config/v1/types_apiserver.go)),
and backend/webhook configuration is an explicit non-goal of the OCP audit policy API
([audit-policy enhancement, Non-Goals](https://raw.githubusercontent.com/openshift/enhancements/master/enhancements/kube-apiserver/audit-policy.md)).

**Supported OCP consumption path.** The Logging operator's `ClusterLogForwarder` has an `audit`
log type covering "the node audit system, `auditd`, Kubernetes API server, OpenShift API
server, and OVN network", routable to Loki, Elasticsearch, Kafka, syslog/Fluentd Forward,
CloudWatch, or Google Cloud Logging
([forwarding docs](https://raw.githubusercontent.com/openshift/openshift-docs/main/modules/cluster-logging-collector-log-forwarding-about.adoc)).
The collector is a per-node DaemonSet (Vector or Fluentd) that tails journald, container logs
and, when configured, the audit sources
([log collection docs](https://raw.githubusercontent.com/openshift/openshift-docs/main/modules/about-log-collection.adoc)).

**Latency.** No published end-to-end figure or SLO exists for the ClusterLogForwarder-to-Loki
path. Component budget: the apiserver's *log* backend writes synchronously by default (batching
is disabled for the log backend — [auditing docs](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/));
Vector's Loki sink flushes a batch at most 1 second old
([Vector Loki sink defaults](https://vector.dev/docs/reference/configuration/sinks/loki/));
Loki ingests over HTTP push. So an in-cluster consumer can realistically see exec audit events
within roughly a second to a few seconds, but sub-second delivery is not guaranteed and degrades
under load. Internal lab observation (Oct 2026, OCP 4.16, not a public source): kube-apiserver
audit events forwarded to Splunk arrived single-digit to ~40 ms after `stageTimestamp` on an
idle cluster, confirming the file-tail path is fast when unloaded — design for seconds, not
milliseconds. A component that needs lower latency would have to tail
`/var/log/kube-apiserver/audit.log` from control-plane nodes itself (privileged hostPath
DaemonSet, outside supported interfaces) or consume a supported push output such as Kafka.

### 3. Mutating admission webhooks on pods/exec

**Is admission invoked, and on what object?** Yes. For connect-style subresources, the apiserver
decodes the connect options object and, if the admission chain handles `Connect`, runs
*mutating* admission and then *validating* admission with operation `CONNECT`, resource `pods`,
subresource `exec`, the `PodExecOptions` as the object, and the requesting user's `userInfo`;
the **mutated object is then handed to `ExecREST.Connect`**
([ConnectResource handler](https://raw.githubusercontent.com/kubernetes/kubernetes/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/rest.go)).
Webhook configurations select these calls with `operations: ["CONNECT"]` and resources
`pods/exec`, `pods/*`, or `*/*`; `oldObject` and `options` are null for CONNECT, and dry-run is
rejected for connect requests
([admission webhook docs](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)).
The AdmissionReview carries its own round-trip `uid` and the full `userInfo`
([admission types](https://raw.githubusercontent.com/kubernetes/api/master/admission/v1/types.go)).

**Does the mutated command reach the runtime process?** Yes. `ExecREST.Connect` builds the
kubelet URL `/exec/{namespace}/{pod}/{container}?command=...&stdin=...` from the mutated
options and proxies to kubelet
([ExecREST / streamLocation](https://raw.githubusercontent.com/kubernetes/kubernetes/master/pkg/registry/core/pod/strategy.go));
kubelet's `getExec` reads the command from the query string and forwards via CRI `Exec`, whose
request contains only `container_id`, `cmd`, `tty`, `stdin`, `stdout`, `stderr`
([kubelet server](https://raw.githubusercontent.com/kubernetes/kubernetes/master/pkg/kubelet/server/server.go);
[CRI api.proto](https://raw.githubusercontent.com/kubernetes/kubernetes/master/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1/api.proto));
CRI-O's OCI handler ultimately runs the OCI runtime's `exec --process` with the (mutated)
argv ([runtime_oci.go](https://raw.githubusercontent.com/cri-o/cri-o/main/internal/oci/runtime_oci.go)).
Important limitation: `PodExecOptions` has **no env field**, so a marker can only reach the
process environment by rewriting argv to run through an `env(1)` wrapper (or by staying in
argv itself).

**Prior art.** The Security Profiles Operator (kubernetes-sigs, installable on OCP) implements
this exact pattern with the `execmetadata.spo.io` mutating webhook: every `kubectl exec` is
rewritten to `env SPO_EXEC_REQUEST_UID=<AdmissionRequest.UID> <command>` (with env-layer
parsing to override existing assignments), the UID is also attached to the request's audit
event as `execmetadata.spo.io/SPO_EXEC_REQUEST_UID`, and a node-side JSON enricher (auditd
reader with an eBPF fallback for short-lived processes) emits `requestUID` in its per-process
JSON audit lines, linking node activity to the API server audit entry that records who started
it ([SPO audit logging guide](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/audit-logging-guide.md);
[execmetadata webhook source](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/webhooks/execmetadata/execmetadata.go)).
SPO also sets the variable on newly added ephemeral containers and on node-debug pods
(`nodedebuggingpod.spo.io`, with OpenShift manifests selecting `oc debug` pods by the
`debug.openshift.io/managed-by: oc-debug` label). The webhooks use the `Ignore` failure policy,
exclude `kube-system`/`kube-public`/`kube-node-lease`/operator namespaces, skip Windows pods,
and exec fails in images without an `env` binary
([security model](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/security-model.md)).
No equivalent ships as a first-party OpenShift platform component.

**Security properties (honest assessment).** SPO's own documentation states: "The attribution
is best effort and not audit evidence. The workload controls the environment of its
processes." Concretely, a container-internal adversary can:

- **Observe:** same-UID processes (and container root) can read `/proc/<pid>/environ`, so the
  marker is visible to anything in the container — including the root process of a *different,
  overlapping* exec session, enabling in-container framing of another session's user. argv is
  world-readable via `/proc/<pid>/cmdline` while the `env` wrapper runs.
- **Forge:** any process may `execve` a new program with an arbitrary `SPO_EXEC_REQUEST_UID`
  value, creating false attribution. The API-server side (audit event + annotation) cannot be
  forged from inside the container.
- **Strip:** the exec root process can unset or overwrite the variable before spawning
  children, making descendants unattributed (false negatives); the audit record itself cannot
  be stripped.

The exec *root* process carries the marker in its environment (the wrapper's argv is transient,
replaced when `env` execve's the target command), and descendants inherit it unless changed.
Consumers must therefore capture the marker at process-exec time and propagate it through
lineage themselves — consistent with the Oct 2026 lab finding that `/proc` polling misses
short-lived processes and a short event buffer (~185 ms skew) is required. The marker-in-audit-
event join key (`AdmissionRequest.UID` ↔ `execmetadata.spo.io/SPO_EXEC_REQUEST_UID`) is the
production-hardened core of the parked admission-based design: the apiserver-side record of
`UID → user` is trustworthy; only the in-container copy is soft.

### 4. Admission for bypass prevention

A validating webhook registered for `CONNECT` on `pods/exec` can reject requests per
user/group/namespace — the validating call runs after the mutating call in the same connect
chain, so it can require that the command was mediated (e.g. starts with the `env` wrapper and
marker) and fail-closed otherwise; an explicit webhook rejection always denies the request
regardless of `failurePolicy` ([connect handler](https://raw.githubusercontent.com/kubernetes/kubernetes/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/rest.go);
[admission docs](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)).
RBAC can independently deny direct exec: subresource permissions are separate from the parent
resource (`"pods"` rules do not match `pods/exec`), so rules with `resources: ["pods/exec"]`
gate exec — but must deny both `create` (SPDY/POST) and `get` (WebSocket/GET), since `get` on
these endpoints is an execute permission ([auditing docs](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/);
[kubelet auth docs warning](https://kubernetes.io/docs/reference/access-authn-authz/kubelet-authn-authz/)).
`oc debug` flows can be gated via `pods/ephemeralcontainers` updates and debug-pod creation.
Hard limits: admission only sees API-server-mediated requests — holders of `nodes/proxy` can
exec in any container on a node via the kubelet API directly, and node SSH / CRI socket access
bypass it entirely ([kubelet auth docs](https://kubernetes.io/docs/reference/access-authn-authz/kubelet-authn-authz/)).
A `Fail` failure policy makes exec unavailable whenever the webhook is down (availability
risk); `Ignore` fails open, which is what SPO chose.

### 5. Node-side metadata

**Kubelet exec flow.** The apiserver proxies the stream to kubelet using its kubelet client
certificate; the transport built from `KubeletClientConfig` forwards no user identity, and
kubelet authorizes the `/exec/...` path as `nodes/proxy` on resource `nodes`
([kubelet client](https://raw.githubusercontent.com/kubernetes/kubernetes/master/pkg/kubelet/client/kubelet_client.go);
[kubelet auth attributes](https://raw.githubusercontent.com/kubernetes/kubernetes/master/pkg/kubelet/server/auth.go)).
Kubelet keeps **no durable session record**: `getExec` resolves the pod, obtains the runtime's
streaming URL, and proxies; only aggregate metrics exist
(`kubelet_http_requests_total` with a `long_running` label covering exec/attach/portforward,
and `kubelet_websocket_streaming_requests_total` by subresource) plus verbosity-gated request
logging ([kubelet server](https://raw.githubusercontent.com/kubernetes/kubernetes/master/pkg/kubelet/server/server.go);
[kubelet metrics](https://raw.githubusercontent.com/kubernetes/kubernetes/master/pkg/kubelet/server/metrics/metrics.go)).

**CRI-O.** The CRI `ExecRequest` carries no user or session context
([api.proto](https://raw.githubusercontent.com/kubernetes/kubernetes/master/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1/api.proto)).
CRI-O's `Exec` returns a streaming URL (its own streaming server, or the conmon-rs container
monitor with WebSocket streaming), and `StreamService.Exec` runs the OCI runtime's
`exec --process` directly, tracking the resulting PID only in memory
(`oci.Container.execPIDs`, deleted on exit) with journald log lines such as "Using exec URL
from container monitor" ([container_exec.go](https://raw.githubusercontent.com/cri-o/cri-o/main/server/container_exec.go);
[container_streaming.go](https://raw.githubusercontent.com/cri-o/cri-o/main/server/container_streaming.go);
[runtime_oci.go](https://raw.githubusercontent.com/cri-o/cri-o/main/internal/oci/runtime_oci.go);
[container.go](https://raw.githubusercontent.com/cri-o/cri-o/main/internal/oci/container.go)).
conmon supervises container init processes (and is used for `ExecSync`), but modern streaming
exec does not spawn a per-exec conmon. OCP ships CRI-O as its only runtime; its NRI plugin
socket covers pod/container lifecycle, not exec sessions. **Nothing on the node knows both the
exec request context (user/auditID) and the resulting PID** — precisely the gap the
`PrivilegedSession` design target closes with an apiserver-issued capability propagated through
kubelet to CRI-O and a node-local attribution agent.

### 6. Host identity limits

**loginuid/auid.** The kernel audit loginuid is set once per login session by PAM at
entry-point applications (login, sshd, gdm, ...); once set it can only be changed with
`CAP_AUDIT_CONTROL`, and kernels expose `only_unset_loginuid`/`loginuid_immutable` features to
lock it down ([pam_loginuid(8)](https://man7.org/linux/man-pages/man8/pam_loginuid.8.html);
[kernel audit.c](https://raw.githubusercontent.com/torvalds/linux/master/kernel/audit.c);
[procfs loginuid](https://raw.githubusercontent.com/torvalds/linux/master/fs/proc/base.c)).
An OpenShift user authenticates at the API server (OAuth token or client cert) and never
creates a PAM session on the node; container processes — including exec'd ones — descend from
systemd → CRI-O → OCI runtime, whose loginuid is unset (4294967295) or a node service's.
Every process in a container therefore shares one meaningless auid, two different OpenShift
users exec'ing into the same container are indistinguishable by auid, and the immutability
rules mean auid cannot be repurposed per-session.

**tlog/SSSD.** `tlog-rec-session` records terminal I/O by acting as the recorded user's login
shell, wrapping the session at login time
([tlog README](https://raw.githubusercontent.com/Scribery/tlog/main/README.md)); SSSD's
`[session_recording]` section (scope `none`/`some`/`all`, plus users/groups) enables it for
host logins on text terminals
([sssd.conf(5)](https://raw.githubusercontent.com/SSSD/sssd/master/src/man/sssd.conf.5.xml)).
`oc exec` creates no host login session — it is a CRI exec inside the container namespaces —
so tlog cannot ride it. (Node SSH/console sessions, e.g. admins debugging nodes, *can* be
recorded this way; useful complementary evidence for the node-access bypass path, not for
container sessions.)

**Node journald.** The node journal contains evidence that an exec occurred — CRI-O streaming
log lines, kubelet request logs at elevated verbosity, long-running-request metrics — and OCP's
logging collects journald from the OS, container runtime, and OpenShift as infrastructure logs
([log collection docs](https://raw.githubusercontent.com/openshift/openshift-docs/main/modules/about-log-collection.adoc));
but none of it carries the requesting user or a durable session record.

## Ranking table

Answers research question 7. Options measured against: user identity, session-to-PID
binding, forgeability by a container-internal adversary, retrofitability on shipped OCP,
and latency.

| # | Option | User identity? | Session→PID binding? | Forgeable by container adversary? | Retrofit on shipped OCP? | Latency |
|---|--------|----------------|----------------------|-----------------------------------|--------------------------|---------|
| 1 | API-server audit events via ClusterLogForwarder (Vector/Fluentd → Loki/Splunk/Kafka) | Yes — authenticated user, groups, credential-id, sourceIPs, UA, command, auditID; OCP login/logout trail | No — time + namespace/pod/container join only; ambiguous under overlapping sessions | Audit record: no (control-plane evidence); the *correlation* is inference, not proof | Yes — supported (APIServer `Default` profile already on; Logging operator) | Event written at request time; delivery seconds-class (Vector batch 1 s default); no published SLO |
| 2 | Mutating admission marker (SPO `execmetadata.spo.io` pattern) | Yes — requestUID join: audit annotation ↔ process env/argv marker; apiserver side not forgeable | Best-effort — marker on exec root env, inherited by descendants; capture needed at exec time | Yes — env is readable/forgeable/strippable in-container; cross-session framing possible; SPO documents this | Yes — SPO installs on OCP today (OLM); no first-party platform component | Marker present at process start (no join latency); audit side seconds-class |
| 3 | Validating admission gating (require mediated exec / deny users) | Indirect — enforcement, not attribution | No | n/a — enforcement control | Yes — plain ValidatingWebhookConfiguration + RBAC | Inline request latency (ms); fail-closed breaks exec on webhook outage |
| 4 | Kubelet / CRI-O node records | No — kubelet sees only the apiserver's client identity (`nodes/proxy`); CRI has no user field | Partial — CRI-O knows exec PID in-memory; no request context, nothing persistent | n/a | Read-only observation today; identity needs upstream change | n/a |
| 5 | Host auid / tlog / SSSD session recording | No for `oc exec` (auid unset/meaningless in containers; tlog only covers host logins) | No | No (auid immutable) but not useful for k8s users | Yes for SSH sessions only | n/a |
| 6 | Upstream audit webhook backend (direct push) | Same as #1 | No | Same as #1 | No — OCP `APIServer` CR has no backend/webhook fields; explicit non-goal | Tunable: batch max-wait (default 30 s) or `blocking` (synchronous) |
| 7 | OpenShift `PrivilegedSession` capability (design target) | Yes — platform-owned record after authn/authz | Yes — signed capability → CRI-O binding → PID + start time + lineage | No — capability never enters the container | No — requires OpenShift platform development (apiserver, kubelet, CRI-O, node agent) | In-stream; no join race beyond process-creation buffer |

## Open questions

- **Productisation:** will Red Hat ship SPO's JSON enricher / exec metadata webhooks as a
  supported OCP component, and in which SPO/OCP versions does the current behaviour
  (env-layer parsing, audit annotations, `oc debug` label) actually land? The docs cited here
  are `main`-branch.
- **Credential-id ↔ token join:** does OCP populate
  `authentication.kubernetes.io/credential-id` for OAuth-token-authenticated exec in a form
  that maps onto `oauthaccesstokens` names (`sha256~...`)? If so, audit events gain a
  token→user confirmation path independent of the username field.
- **Quantified pipeline latency:** no public end-to-end numbers exist for
  ClusterLogForwarder → Loki under production API-server load; only component defaults
  (Vector 1 s batch) are documented. Worth measuring on a loaded cluster before promising
  ~1 s correlation.
- **Operational acceptability of fail-closed exec:** a validating webhook requiring the
  mediation marker (`failurePolicy: Fail`) denies all exec during webhook outages — needs an
  operational decision plus break-glass flow.
- **Marker stripping detection:** should unattributed process activity during an active
  session be treated as suspicious (marker stripped) rather than merely unattributed, and can
  the eBPF-based capture make stripping observable?
- **Attach sessions:** admission cannot add a marker to attach (`PodAttachOptions` has no
  command); bounding attach remains gating-only or requires stream-level capture at
  kubelet/CRI-O.
- **Verb skew:** which OCP 4.x releases enable `AuthorizePodWebsocketUpgradeCreatePermission`
  (WebSocket exec authorized as `get` vs `create`), and do audit-based detections need to
  accept both verbs for `pods/exec`?
