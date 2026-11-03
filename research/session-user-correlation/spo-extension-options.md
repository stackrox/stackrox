# SPO Extension Options for Session-User Correlation

Research note evaluating the Kubernetes Security Profiles Operator (SPO) as a home for
privileged-session identity binding and checks, in support of
`PRIVILEGED_SESSION_ATTRIBUTION_RESEARCH.md` (attributing runtime file activity to the
authenticated OpenShift user behind an `oc exec` session, with unforgeable evidence).

Scope reminder: SPO's enrichers enrich kernel AUDIT events (seccomp/SELinux/AppArmor
denials or logged syscalls) — they are not a file-activity telemetry source. Collector/FACT
keeps that role; this note only evaluates SPO as the home for the session-identity join and
privileged session checks.

## Summary

SPO already ships an opt-in, best-effort session-identity join: the `execmetadata.spo.io`
and `nodedebuggingpod.spo.io` mutating webhooks inject `SPO_EXEC_REQUEST_UID` into exec
commands, debug pods, and ephemeral containers, and the JSON enricher emits it per process
as `requestUID` for correlation with the API-server audit log. This is exactly the
"SPO-assisted flow" our earlier exploration sketched — it landed upstream in 2025 after a
privacy review reduced it to an opaque request UID (no username). SPO's own documentation
states the attribution is forgeable by a container-internal adversary and is "not audit
evidence", which matches our lab findings. SPO is architecturally well suited to HOST a
node-local session-binding agent (per-node privileged daemon, PID+start-time caches, a BPF
exec-observation path that avoids the /proc race, node-local gRPC socket precedents), but
it cannot OWN the trust anchors the design doc requires: signed capabilities, CRI-O
reporting, and an audit-grade `PrivilegedSession` record are control-plane surfaces SPO has
no interface to, and OpenShift's supported audit configuration cannot feed an in-cluster
webhook sink. Verdict: viable home for the correlation/join mechanics and the node agent;
not a viable home for forensic-grade identity issuance without an OpenShift-owned
control-plane component feeding it.

## Findings

### 1. SPO architecture today

SPO consists of three components in the operator namespace
([doc/README.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/README.md)):

- **Manager** (Deployment `security-profiles-operator`): watches the SPOD configuration,
  creates the daemon and webhook from it, merges recorded profiles, aggregates per-node
  profile state, and deletes seccomp profiles rejected by the SPOD allow lists.
- **Daemon** (DaemonSet `spod`): runs on every node; installs seccomp/AppArmor/SELinux
  profiles (SELinux through `selinuxd`) and reports state as `SecurityProfileNodeStatus`.
  Optional containers: `bpf-recorder`, `log-enricher`, `json-enricher`.
- **Webhook** (Deployment `security-profiles-operator-webhook`): applies `ProfileBinding`
  profiles to pods, marks `ProfileRecording` pods for recording, adds the metadata the
  JSON enricher needs for `kubectl exec` and node debugging sessions, and validates
  `RawSelinuxProfile`.

CRDs: `SeccompProfile` (sp), `AppArmorProfile` (aa), `SelinuxProfile` (selp),
`RawSelinuxProfile` (rselp), `ProfileBinding` (pb), `ProfileRecording` (prec),
`SecurityProfilesOperatorDaemon` (spod), `SecurityProfileNodeStatus` (spns)
([doc/README.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/README.md)).
Configuration is a single SPOD object named `spod`; feature switches include
`enableAppArmor`, `selinux.enable` (default on OpenShift), `enricher.enableLogEnricher`,
`enricher.enableBpfRecorder`, `enricher.enableJsonEnricher`, and
`enricher.enableExecMetadata`
([doc/configuration.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/configuration.md)).

Privileges ([doc/security-model.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/security-model.md)):

| Feature | Root | HostPID | Privileged | Host mounts |
|---|---|---|---|---|
| Seccomp install | No (non-root enabler init) | No | No | `/var/lib/security-profiles-operator`, kubelet seccomp dir |
| SELinux | Yes | No | init container + reload Job | `/etc/selinux`, `/var/lib/selinux`, `/sys/fs/selinux` |
| AppArmor | Yes | Yes | Yes | host mount namespace |
| Log enricher | Yes | Yes | No (SYS_PTRACE, DAC_READ_SEARCH, CHOWN) | `/var/log/audit`, `/var/log` (read-only) |
| JSON enricher | Yes | Yes | No | `/var/log`, `/sys/kernel/debug`, `/sys/kernel/tracing` |
| BPF recorder | Yes | Yes | No | `/sys/kernel/debug`, `/sys/kernel/security`, `/sys/kernel/tracing`, `/sys/fs/cgroup` |

On OpenShift, only the pods of the `spod` service account (the `spod` DaemonSet and the
SELinux policy reload Job) use the `privileged` SCC; the operator and webhook deployments
run non-root and are pinned to `restricted-v2` via the `openshift.io/required-scc`
annotation
([doc/security-model.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/security-model.md)).

Relevant precedent for a node-local agent: the daemon containers already expose
node-local Unix-domain gRPC servers — `/var/run/grpc/metrics.sock`,
`/var/run/grpc/enricher.sock`, `/var/run/grpc/bpf-recorder.sock`
([internal/pkg/config/config.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/config/config.go)),
and the log/JSON enrichers talk to them over `daemongrpc`
([enricher.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/enricher/enricher.go)).

### 2. Deployment on OpenShift

- "OpenShift ships the operator in its Red Hat catalog, install it from the operator
  catalog of the web console" — i.e., it is a Red Hat-published, supported operator on
  OCP, not merely community-installed
  ([doc/installation.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/installation.md)).
  Upstream installs on OCP are also possible via operatorhub.io or the upstream catalog
  (`CatalogSource` in `openshift-marketplace`).
- OpenShift needs no cert-manager (the service CA injects webhook certificates), and the
  OLM CSV carries `operators.openshift.io/valid-subscription:
  ["OpenShift Kubernetes Engine", "OpenShift Container Platform", "OpenShift Platform Plus"]`
  ([CSV 0.10.1](https://raw.githubusercontent.com/k8s-operatorhub/community-operators/main/operators/security-profiles-operator/0.10.1/manifests/security-profiles-operator.clusterserviceversion.yaml)).
- Requirements: Kubernetes v1.30 or later
  ([doc/installation.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/installation.md)).
  Namespace: `security-profiles-operator` (suggested namespace in the CSV; must allow
  privileged pods for the daemon).
- SCCs: the CSV grants `securitycontextconstraints use` to the operator, `spo-webhook`,
  and spod service accounts
  ([CSV 0.10.1](https://raw.githubusercontent.com/k8s-operatorhub/community-operators/main/operators/security-profiles-operator/0.10.1/manifests/security-profiles-operator.clusterserviceversion.yaml));
  in practice only `spod`-service-account pods run with the `privileged` SCC
  ([doc/security-model.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/security-model.md)).
  SELinux recording additionally requires workloads to use an SCC with
  `seLinuxContext: RunAsAny`
  ([doc/troubleshooting.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/troubleshooting.md)).
- SELinux support is enabled by default on OpenShift
  ([doc/configuration.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/configuration.md)),
  and the repo carries downstream OpenShift manifests
  (`deploy/openshift-dev.yaml`, `deploy/openshift-downstream.yaml`).
- Red Hat actively maintains the OCP packaging: JSON-enricher fixes landed under
  OCPBUGS-62730
  ([PR #3071](https://github.com/kubernetes-sigs/security-profiles-operator/pull/3071)),
  and the OCP node user-activity-auditing program references OCPSTRAT-846/OCPNODE-3426
  ([openshift/oc PR #2055](https://github.com/openshift/oc/pull/2055)).

### 3. Log-enricher internals

- **Input source**: the log enricher *tails a file*, it does not use audit netlink. The
  default `Auditd` source follows `/var/log/audit/audit.log` with a syslog fallback
  (`/var/log/syslog`); `enricher.logEnricherSource: Bpf` switches to an eBPF source that
  reads AppArmor audit events directly from the kernel (Linux ≥ 5.19, AppArmor events
  only)
  ([doc/recording.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/recording.md),
  [doc/configuration.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/configuration.md),
  [auditsource/audit.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/enricher/auditsource/audit.go)).
  It parses only SECCOMP (type 1326), AVC (1400), and APPARMOR_* record types; everything
  else is dropped (same source).
- **Correlation with workloads**: audit-line PID → `util.ContainerIDForPID` reads
  `/proc/<pid>/cgroup` to extract the 64-digit container ID (and `/proc/<pid>/stat` for the
  process start time)
  ([util/containers.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/util/containers.go))
  → the container ID is resolved to pod/namespace/container via a cluster-scoped pod watch
  (`podindex`), with a TTL cache and per-container backlogs while the pod status has not
  listed the container yet (up to 1 minute)
  ([container.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/enricher/container.go),
  [lookup.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/enricher/lookup.go)).
  PID reuse is handled explicitly: entries are keyed by PID + start time, the container of
  an exited process is retained for one minute, and lines whose PID started after the
  audit timestamp are dropped (`errProcessStartedLater`) (lookup.go).
- **Output**: a structured log line `"audit"` with key/value pairs — timestamp, type,
  node, namespace, pod, container, executable, pid, syscallID, syscallName, arch (seccomp)
  or apparmor/operation/profile/name/extra_info (AppArmor/SELinux AVCs) — written to the
  container log; audit metrics are sent over the node-local gRPC socket; and recorded
  syscall/AVC sets are served to the profile recorder over the enricher gRPC API
  ([enricher.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/enricher/enricher.go)).
- **Identity fields in enriched output**: uid/gid (parsed from the audit record's `uid=`
  / `gid=` fields, only for seccomp lines; nil when absent), plus the SELinux contexts
  (`scontext`/`tcontext`/`tclass`) or AppArmor fields. There is **no `auid` (audit UID)
  parsing anywhere** in the audit-line extraction
  ([types.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/enricher/types/types.go),
  [audit.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/enricher/auditsource/audit.go)).
  So the enricher reports the *container-internal* uid (usually root), not the
  `oc`/`kubectl` requester. Log-based recording has no effect at all for privileged
  containers, which are not subject to seccomp/SELinux policies
  ([doc/recording.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/recording.md)).

### 4. eBPF recorder

- The BPF recorder records `SeccompProfile` (syscalls) and `AppArmorProfile` (file,
  socket, capability events) for selected workloads; it requires a kernel with BTF at
  `/sys/kernel/btf/vmlinux`, and AppArmor recording additionally requires the BPF LSM
  ([doc/recording.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/recording.md)).
- **Keying**: events carry a header of `{pid, mntns, key, type, flags}` where "Key is the
  key the recorded data is stored under, the cgroup ID or the mount namespace" — seccomp
  syscalls are recorded per mount namespace (the `mntns_syscalls` map), AppArmor events
  per cgroup ID
  ([bpfrecorder.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/bpfrecorder/bpfrecorder.go)).
  Keys are mapped to container IDs via the pod status (with a container lookup timeout and
  stale-key sweeps) (same source).
- **Process lineage**: limited, in-recording only. The recorder maintains `activePids`
  and `childPids` BPF maps to propagate a container's recording key from parent to child
  processes during a recording session, and detects container init processes by their
  `comm` (`runc:[2:INIT]`, `youki:[2:INIT]`, `crun`, `memfd:crun_clon...`) (same source).
  There is no user or session identity, and no durable lineage beyond a recording.
- **Separate and highly relevant**: the BPF *process cache* used by the JSON enricher
  loads a BPF program on the `sched_process_exec` tracepoint and captures each exec's
  arguments **and environment** (up to 4096/8192 bytes), from which it extracts *only*
  `SPO_EXEC_REQUEST_UID` — "The rest of the environment is not kept, it can carry
  credentials" — plus the process start time, and it detects PID replacement
  ([bpfprocesscache.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/bpfrecorder/bpfprocesscache.go)).
  Because it observes exec at the tracepoint rather than reading `/proc` after the fact,
  it is the mechanism SPO uses to close the short-lived-process gap that our lab saw with
  `/proc` polling
  ([doc/audit-logging-guide.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/audit-logging-guide.md)).

### 5. Extension points for privileged session checks

**Could spod host a node-local session-to-(host PID, PID start time) binding agent?**
Mechanically yes, and it is a natural fit with existing patterns:

- The daemon already runs optional, SPOD-gated containers per node (log-enricher,
  bpf-recorder, json-enricher) that require hostPID and node mounts — a
  `session-binding-agent` container would follow the same model
  ([doc/README.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/README.md),
  [doc/configuration.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/configuration.md)).
- Its process machinery already keys processes by **(PID, PID start time)** —
  `processContainer{id, started}`, cache keys `"<pid>_<starttime>"`, and the BPF process
  cache's `StartedAt` — which is precisely the binding key the design doc proposes
  ([lookup.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/enricher/lookup.go),
  [process.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/enricher/process.go),
  [bpfprocesscache.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/daemon/bpfrecorder/bpfprocesscache.go)).
- Node-local, read-only gRPC APIs on Unix sockets are an established SPO pattern
  (`/var/run/grpc/*.sock`, [config.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/config/config.go)),
  matching the design doc's Session Attribution API transport choice.

**How could spod LEARN exec sessions, given it has no API-server audit visibility?**

- **(a) OCP audit webhook sink consumed in-cluster — not viable on OpenShift.** Upstream
  kube-apiserver supports webhook audit backends via `--audit-webhook-config-file`
  ([kubernetes.io audit docs](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/)),
  but OpenShift's supported configuration surface, the `APIServer` CR
  (`config.openshift.io/v1`), exposes only the preset profiles `Default`,
  `WriteRequestBodies`, `AllRequestBodies`, `None` plus per-group custom rules — no
  custom policy file and no webhook backend
  ([openshift/api types_apiserver.go](https://raw.githubusercontent.com/openshift/api/master/config/v1/types_apiserver.go)).
  Shipping the API audit *log files* off-node (e.g. via cluster log forwarding) and
  joining centrally is possible but would not feed a node-local agent through a supported
  API.
- **(b) CRD populated by a control-plane component — plausible consumer, not producer.**
  SPO watches cluster-scoped pods already (podindex) and could watch a
  `PrivilegedSession`-style CRD, but the record must be written by something with
  control-plane trust (per the design doc, the API server itself). SPO has no precedent
  for owning an authorisation-grade API; its CRDs are configuration/state objects, and
  labels/annotations are explicitly not forensic evidence (they are user-mutable). The
  nearest existing pattern is that the exec-metadata webhooks stamp the request UID as an
  **audit annotation** on the API-server audit event
  (`execmetadata.spo.io/SPO_EXEC_REQUEST_UID`), which is a control-plane-visible join key
  but not a durable, platform-owned session record
  ([doc/audit-logging-guide.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/audit-logging-guide.md)).
- **(c) Mutating admission on pods/exec (the prior parked design) — this is now upstream
  SPO, shipped.** The `execmetadata.spo.io` webhook intercepts CREATE/CONNECT on
  `pods/exec` and `pods/ephemeralcontainers` and rewrites `PodExecOptions.command` to run
  through the container's `env` binary with `SPO_EXEC_REQUEST_UID=<request UID>`;
  `nodedebuggingpod.spo.io` sets the variable in node-debug pods (selected by the
  `app.kubernetes.io/managed-by: kubectl-debug` label) and in newly created ephemeral
  containers
  ([bindata/webhook.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/manager/spod/bindata/webhook.go),
  [execmetadata.go](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/internal/pkg/webhooks/execmetadata/execmetadata.go),
  [doc/audit-logging-guide.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/audit-logging-guide.md)).
  The webhook adds the request UID as an audit annotation, skips Windows pods, fails open
  with warnings, and parses layered `env` command prefixes to survive `env -i`/`-u`
  attempts (execmetadata.go). SPO's own documentation states the limit plainly: "The
  attribution is best effort and not audit evidence. The workload controls the environment
  of its processes: a pod author can set the variable on a container, so that its
  processes carry a made-up request UID, and a process in an exec session can unset or
  change it before it starts other programs"
  ([doc/audit-logging-guide.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/audit-logging-guide.md)).
  The JSON enricher then reads the request UID from `/proc/<pid>/environ` (or from the
  `env` command's arguments) with the BPF process cache as fallback, and emits it as
  `requestUID` in each JSON line, for correlation with the API-server audit log (same
  doc). Note `pods/attach` is **not** intercepted — attach remains unattributed,
  consistent with our design doc's scope.
- **(d) Direct kubelet/CRI-O interfaces — none exist in SPO.** SPO's node-side data
  sources are `/proc`, eBPF, and the Kubernetes API pod watch; it never talks to the kubelet
  or CRI-O sockets. The design doc's trusted path (kubelet validates a signed capability
  and CRI-O reports the created process through an OpenShift-specific extension) requires
  changes in kubelet/CRI-O that are outside SPO's repository and governance. SPO's closest
  analogue is comm-based detection of runtime init processes and the
  `sched_process_exec` BPF capture, which can observe *that* a process was created but
  cannot bind it to an authorised API request without a trusted join key from the
  control plane.
- **(e) Other levers found**: client-side labelling of debug pods extends webhook reach
  (`oc debug node/...` pods are labelled `debug.openshift.io/managed-by: oc-debug` —
  [openshift/oc PR #2055](https://github.com/openshift/oc/pull/2055), fixing the
  objectSelector problem described in
  [doc/audit-logging-guide.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/audit-logging-guide.md);
  kubectl gained an equivalent label via
  [kubernetes/kubernetes#131791](https://github.com/kubernetes/kubernetes/pull/131791)).
  Enricher log filtering (`jsonEnricherFilters`, `logEnricherFilters`) can reduce JSON
  output to exec-session lines only
  ([doc/audit-logging-guide.md](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/doc/audit-logging-guide.md)).

### 6. Existing SPO work on user identity, request identity, sessions

There is a full, active feature line — driven by Red Hat's OpenShift user-activity
auditing program — culminating in the shipped exec-metadata webhooks:

- [#2817 "Add In-Pod Activity Audit Recorder"](https://github.com/kubernetes-sigs/security-profiles-operator/issues/2817)
  (2025-04, ngopalak-redhat): the JSON enricher's origin; motivation is audit of SRE
  activity inside pods, interleavable with Kubernetes audit logs.
- [#2857 "Add end user information to In-Pod Activity logs"](https://github.com/kubernetes-sigs/security-profiles-operator/issues/2857)
  (2025-05, ngopalak-redhat): the request-UID origin. It explicitly considered and
  rejected "sending user information directly from the Kubernetes API server to Kubelet"
  as bad core-Kubernetes design, and originally proposed injecting both REQUEST_UID and
  REQUEST_USERNAME.
- [PR #2878](https://github.com/kubernetes-sigs/security-profiles-operator/pull/2878)
  (merged 2025-06, size/XXL): implemented the mutating webhook + JSON enricher request
  UID. The privacy review outcome is recorded in the PR: "Initial PR was raised to inject
  the User information. Since there are concerns of PII now only the request UID is
  injected. Also made the webhook optional."
- Follow-ups: Ubuntu fixes
  ([#2893](https://github.com/kubernetes-sigs/security-profiles-operator/pull/2893)),
  kubectl-debug handling ([#2896](https://github.com/kubernetes-sigs/security-profiles-operator/pull/2896)),
  node-debug request-UID injection fix
  ([#2907](https://github.com/kubernetes-sigs/security-profiles-operator/pull/2907)),
  OpenShift-only webhook options for the node-debug label
  ([#2966](https://github.com/kubernetes-sigs/security-profiles-operator/pull/2966)),
  OCP-bundled fixes under OCPBUGS-62730
  ([#3071](https://github.com/kubernetes-sigs/security-profiles-operator/pull/3071)),
  and ongoing e2e hardening under OCPNODE-4767
  ([#3496](https://github.com/kubernetes-sigs/security-profiles-operator/pull/3496), open
  as of Nov 2026).
- Adjacent OpenShift work targeting OCP 4.20: the `oc debug` pod label for reliable
  webhook targeting, under OCPSTRAT-846 (user requirement: user activity auditing for
  OpenShift nodes) and OCPNODE-3426
  ([openshift/oc PR #2055](https://github.com/openshift/oc/pull/2055), plus fix
  [#2074](https://github.com/openshift/oc/pull/2074)).
- Ongoing recorder hardening continues on main (e.g. exec-args capture in
  `sched_process_exec` [#3648](https://github.com/kubernetes-sigs/security-profiles-operator/pull/3648),
  env-command exec fix [#3646](https://github.com/kubernetes-sigs/security-profiles-operator/pull/3646),
  both merged end of Oct 2026).

Our parked "SPO-assisted flow" (env-var request UID captured by Collector/FACT at exec
time, propagated through lineage) is therefore not hypothetical: the injection side is
upstream, documented, and Red-Hat-maintained; the privacy question was settled by
reducing the payload to the opaque request UID, with the username resolved only through
the API-server audit log join.

### 7. Upstream feasibility

- **Governance**: `kubernetes-sigs/security-profiles-operator` is a SIG-Node subproject
  ([kubernetes/community sig-node README](https://raw.githubusercontent.com/kubernetes/community/master/sig-node/README.md)),
  Apache-2.0, with a monthly community meeting and an OpenSSF best-practices badge
  ([README](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/README.md)).
- **Maintainers**: approvers `ccojocar`, `saschagrunert`, `JAORMX`, `jhrozek`; reviewers
  `Vincent056`, `ngopalak-redhat`
  ([OWNERS](https://raw.githubusercontent.com/kubernetes-sigs/security-profiles-operator/main/OWNERS)).
  Red Hat involvement is strong: the request-UID/JSON-enricher line is authored by
  `ngopalak-redhat`, maintainer commits are signed with `@redhat.com` addresses (e.g. the
  2026 docs restructure by Sascha Grunert), and the OCP delivery is tracked with Red Hat
  Jira references (OCPSTRAT/OCPNODE/OCPBUGS) in SPO PRs.
- **Activity**: healthy and current — releases v0.10.0 (2025-08), v1.0.0 (2026-06),
  v1.1.0 (2026-09), v1.1.1 (2026-10), v1.1.2 (2026-10-29); repository last pushed
  2026-11-03; 879 stars, 5 open issues
  ([GitHub releases](https://github.com/kubernetes-sigs/security-profiles-operator/releases),
  [GitHub API](https://api.github.com/repos/kubernetes-sigs/security-profiles-operator)).
- **Precedents for sizeable features**: the eBPF recorder (large C + Go feature,
  [talk](https://youtu.be/3dysej_Ydcw) 2022), AppArmor support (privileged daemon mode),
  the `spoc` CLI, OCI base-profile distribution
  ([KEP-6061](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6061-oci-artifact-security-profiles)),
  and the JSON enricher itself (issue #2817 → PR #2878 (XXL) → released in v0.10.0, with
  follow-up fixes continuing for over a year).
- **Effort estimate** for a session-binding/enrichment feature: the *forgeable* variant
  (webhook injection + enricher join + BPF process cache + e2e) took roughly one quarter
  from issue to release (Apr 2025 → Aug 2025) by effectively one Red Hat engineer plus
  maintainers. A *trusted* variant matching the design doc (node-local binding agent
  container in spod + node-local watch API + a session CRD consumer + hardening/e2e)
  would be SPO-sized at several engineer-months, assuming the trust inputs (signed
  capabilities, CRI-O exec reporting) arrive from OpenShift components; those
  kubelet/CRI-O/control-plane changes are the long pole and sit outside SPO entirely.

## Assessment

**Is SPO a viable home for privileged session checks?** For the *join* — yes, and it is
already the de-facto upstream home: the request-UID join (exec metadata webhooks →
process env/BPF capture → `requestUID` in JSON lines → API-server audit correlation) is
shipped, opt-in via `enricher.enableJsonEnricher`, Red-Hat-maintained, and targeted at
the OCP 4.20 user-activity-auditing requirement. For ACS, consuming this join is
low-hanging fruit: Sensor can correlate the SPO `requestUID` on runtime events with the
API-server audit event (the same lookup our admission-identity LRU prototype built), and
the BPF process cache answers our lab's short-lived-process gap.

**What fits SPO's scope:**

- Hosting the node-local binding agent as an optional spod container (DaemonSet,
  hostPID, SPOD feature switch, node-local gRPC socket — all established patterns).
- The (host PID, PID start time) keying, PID-reuse defences, and exec-time BPF capture.
- Best-effort enrichment of its own audit output with session/request identifiers.
- Watching a platform-provided session CRD and exposing bindings locally.

**What strains or breaks SPO's scope:**

- **Forensic-grade trust.** SPO's own docs classify the env-var binding as forgeable
  ("best effort and not audit evidence"). Signed session capabilities, nonce consumption,
  and a platform-owned `PrivilegedSession` record require API-server/kubelet/CRI-O
  changes and key management that a profile-management operator cannot own. SPO could
  *consume* such trust anchors but never *issue* them.
- **Learning sessions without audit visibility.** The two candidate feeds are weak on
  OpenShift: audit webhook sinks are not exposed by the supported `APIServer`
  configuration, and a control-plane-populated CRD presupposes the OpenShift component
  the design doc already calls for. Admission mutation (the shipped path) cannot be made
  unforgeable from inside the workload's environment.
- **No kubelet/CRI-O interface** exists in SPO today, and adding one is a cross-repo
  (cri-o, kubernetes) effort, not an SPO change.
- **Bypass prevention** (restricting direct `pods/exec`, node SSH, CRI socket access to
  mediated pathways) is admission/RBAC/SCC policy owned by the platform, not SPO.
- **Privacy posture**: upstream settled on an opaque request UID precisely to avoid
  putting usernames in workload-visible environments; any ACS design must keep the
  username resolution on the control-plane side (audit log/PrivilegedSession), which our
  design doc already does.

**When a separate operator would be cleaner:** if the deliverable is the *trusted access
pathway* of the design doc — signed capabilities, CRI-O reporting, platform-owned session
records, and enforcement that unmediated access is blocked — that is an OpenShift
platform component (or a purpose-built operator with control-plane integration), not SPO.
SPO remains attractive as the *delivery vehicle for the node agent* if the platform feeds
it trusted bindings (CRD or populated BPF map), because the daemon plumbing, PID+start
time machinery, and node-local API patterns already exist there. A pragmatic split:
OpenShift owns identity issuance and CRI-O/kubelet propagation; SPO (or a fork of its
daemon patterns) hosts the node-local binding table and read API; ACS/FACT/Collector
consume bindings read-only and keep file telemetry.

## Open questions

1. Would SPO maintainers accept a session-binding agent container, or consider it scope
   creep beyond profile management? (The JSON-enricher precedent suggests appetite for
   audit-adjacent features driven by Red Hat requirements, but a trusted-binding agent
   implies a new CRD and control-plane contract.)
2. Who writes the session CRD in an OCP delivery — an OpenShift control-plane component
   (per the design doc) or an SPO-side controller? What RBAC would spod need to watch
   exec-session records cluster-wide?
3. CRI-O reporting interface: does any usable hook or extension exist today for
   reporting (container ID, exec ID, PID, start time) at exec time, and what is the
   realistic upstream path in cri-o for the design doc's trusted propagation?
4. Timing: does SPO's `sched_process_exec` BPF capture close the ~185 ms binding lag our
   lab measured with `/proc`-based correlation? Needs a measurement; the code suggests
   yes for exec-args/request-UID capture (tracepoint fires at exec), but the container-ID
   resolution still goes through the pod watch.
5. Attach sessions: SPO intercepts `pods/exec` and `pods/ephemeralcontainers` only;
   `pods/attach` remains uncovered — confirm ACS keeps attach as best-effort/unattributed
   (consistent with the design doc).
6. GA readiness: the JSON enricher's log format is still `spo/v1_alpha`, its advanced
   e2e tests are WIP (OCPNODE-4767 / PR #3496, open as of Nov 2026), and the enricher
   requires workload seccomp profiles that log syscalls — what is the OCP 4.20 delivery
   state of `enableJsonEnricher` and the oc-debug label integration?
7. For ACS consumption: is the JSON enricher's node-local output (stdout/audit log file)
   the right ingestion point for Sensor, versus FACT reading the daemon's gRPC enricher
   socket or a future binding API directly?
