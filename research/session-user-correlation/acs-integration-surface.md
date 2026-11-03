# ACS Integration Surface for Session-User Correlation

Research note for TaskWarrior task `ccfef26d-f70f-47e8-8ba7-3b5647731ab1`.
Scope: inventory of the ACS (StackRox) side of attributing runtime file activity to
the authenticated OpenShift user behind an `oc exec`/`oc debug` session, compared
against the design target in `PRIVILEGED_SESSION_ATTRIBUTION_RESEARCH.md` (main
checkout, untracked). All prototype claims are read-only observations of local
branches `giles/k8s-fam-correlation` (commit `2331c00b19`) and
`giles/spo-uid-tracking` (commit `f9324ed3a2`); no prototype code is merged or
checked out here.

## Summary

Master already has the full file-activity signal path (Collector/FACT -> Sensor
filesystem pipeline -> detector -> alert -> Central -> UI), user-aware kube-event
parsing (`pkg/kubernetes/event.go`), an admission-control webhook that sees
exec/attach Connect requests, and node-side audit-log ingestion. Two prototype
branches cover complementary halves: `giles/k8s-fam-correlation` builds the
audit-log session tracker and enriches `FileAccess` with an
`InteractiveSession` (user, audit ID, confidence, match basis) — but hardcodes
"high" confidence, keys on mutable names, and has no metrics or restart story;
`giles/spo-uid-tracking` builds an admission-request-UID -> user cache in Sensor
fed by the admission controller, but nothing consumes it. Neither prototype
implements the design target's verified session-ID binding (no session field on
process signals, no FACT/Collector tagging, no durable PrivilegedSession
record, no matchBasis taxonomy). The Splunk demo and exec-audit lab show the
join is feasible offline and that PID-lineage binding works but arrives after
file writes and can miss short-lived processes.

## Findings

### 1. Prototype `giles/k8s-fam-correlation` (commit `2331c00b19`, 24 files)

Inspected via `git show 2331c00b19` / `git diff master...giles/k8s-fam-correlation`
only; branch never checked out. Master has no pod exec/attach audit events today
(`compliance/collection/auditlog/auditlog_impl.go` allow-lists only
`ResponseComplete`/`Panic` stages and secret/config/rbac/network resources).

**Audit-log lifecycle parsing (compliance/collection/auditlog/):**

- `auditevent.go`: adds `pods_attach` to the resource map; parses
  `objectRef.subresource` and builds a `resourceKey()` of
  `resource + "_" + subresource`; `ToKubernetesEvent` sets the new
  `AuditStage` and populates `PodExecArgs{container, commands}` /
  `PodAttachArgs{container}` by parsing the audit event's `requestURI` query
  parameters (`requestQueryValue`/`requestQueryValues` using
  `net/url.ParseRequestURI`).
- `auditlog_impl.go`: splits the stage allow-list —
  `completionStagesAllowList` (`ResponseComplete`, `Panic`) for existing
  resources and `interactiveStagesAllowList` (adds `ResponseStarted`) for
  `pods_exec`/`pods_attach`, both newly added to `resourceTypesAllowList` with
  empty verb deny-lists. This preserves the exec/attach lifecycle: session
  starts at `ResponseStarted`, ends at `ResponseComplete` or `Panic`.
- Tests: `auditlog_impl_test.go`
  (`TestReaderSendsInteractiveAuditLifecycleEvents`), `sender_impl_test.go`
  (`TestSendPreservesInteractiveAuditLifecycle`, asserts container `main` and
  commands `[sh, -c]` parsed from the request URI).

**Session tracker (`sensor/common/interactivesessions/tracker.go`, new, 133 lines):**

- `Tracker` holds `map[key]map[string]session` where `key` is
  `(namespace, pod, container)` and the inner map is keyed by audit ID
  (the `KubernetesEvent.id` carries the audit ID from
  `compliance/collection/auditlog/auditevent.go`).
- `Observe(*sensor.AuditEvents)`: prunes expired entries, adds a session on
  `ResponseStarted`, deletes by audit ID on any other stage
  (`ResponseComplete`/`Panic` end the session).
- `Match(namespace, pod, container)` returns the event **only when exactly one
  session is active for the key** — ambiguous concurrency yields no attribution.
- Bounds: `defaultTTL = 5 * time.Minute`, `maxSessions = 1024` (new sessions
  silently dropped at capacity; re-observing an existing audit ID refreshes
  expiry). Tests in `tracker_test.go` cover single-session match,
  `ResponseComplete` removal, ambiguous concurrent sessions, and TTL expiry.

**Filesystem pipeline (`sensor/common/filesystem/pipeline/pipeline.go`):**

- `NewFileSystemPipeline` gains a `sessionTracker` parameter; in
  `translateWithIndicator`, after building the `storage.FileAccess`, it calls
  `Match(indicator.GetNamespace(), indicator.GetPodId(),
  indicator.GetContainerName())` and sets `access.InteractiveSession`.
- Helper `interactiveSession(event)` builds
  `storage.FileAccess_InteractiveSession{EffectiveUser, AuthenticatedUser,
  AuditId, Type, Confidence, MatchBasis}` — effective user prefers the
  impersonated user, `type` is `"exec"`/`"attach"`, `confidence` is hardcoded
  `"high"`, `matchBasis` is hardcoded `"namespace/pod/container"`.

**Protos:**

- `proto/storage/file_access.proto`: new `FileAccess.InteractiveSession`
  message (`effective_user`, `authenticated_user`, `audit_id`, `type`,
  `confidence`, `match_basis` — all free-form strings) and
  `FileAccess.interactive_session = 7`. Generated code committed
  (`generated/storage/file_access.pb.go`, `file_access_vtproto.pb.go`).
- `proto/storage/kube_event.proto`: adds `audit_stage = 22` (commented "will
  not be used for policy detection"); `PodExecArgs`/`PodAttachArgs` already
  exist on master.

**Detector (`sensor/common/detector/detector.go`):** new
`auditEventsForDetection()` filters `ResponseStarted` events out of the audit
batch before `DetectAuditLogEvents` so lifecycle events do not double-fire
k8s-event policies (test in `detector_audit_log_test.go`).

**Wiring:** `sensor/kubernetes/sensor/sensor.go` creates one
`interactivesessions.New()` and passes it to both `compliance.NewService` and
`NewFileSystemPipeline`; `sensor/common/compliance/service_impl.go` `Communicate`
feeds every `MsgFromCompliance_AuditEvents` batch into
`sessionTracker.Observe` (nil-guarded).

**UI/GraphQL:** `central/graphql/resolvers/generated.go` registers
`FileAccess_InteractiveSession` (+86 lines); `ui/apps/platform/src/types/
fileAccess.proto.ts` adds the TS type; `ui/apps/platform/src/Containers/
Violations/Details/FileAccessCardContent.tsx` renders "Kubernetes actor",
"Interactive session", and
`Attribution: high confidence (namespace/pod/container)` description-list
items.

**Done vs missing against the stated brief (a) bounded tracker keyed by audit
ID with TTL expiry, immutable container-ID resolution from names, metrics,
restart semantics:**

- Bounded + TTL + audit-ID keying: **done** (1024 sessions, 5-minute TTL,
  inner map keyed by audit ID; graceful capacity drop).
- Immutable container-ID resolution from names: **missing** — `Match` joins on
  `(namespace, pod name, container name)` strings taken from the audit
  request URI and the enriched indicator. There is no resolution to container
  ID or Pod UID, so a pod deleted and its name reused within the TTL can match
  against the wrong generation, and container-name mismatches (audit query
  parameter absent -> `key.container == ""` -> entry skipped) silently
  un-attribute.
- Metrics: **missing** — no Prometheus counters/gauges for active sessions,
  drops at capacity, match hits/misses, or expired sessions.
- Restart semantics: **missing** — the tracker is in-memory only. Sensor
  restart loses all sessions; the audit-log tailer resumes from its persisted
  offset (see question 7), so pre-restart `ResponseStarted` events are never
  replayed and the enrichment gap lasts until sessions end.

**Done vs missing against the stated brief (b) FileAccess results exposing
user plus confidence and match basis, with no attribution under ambiguous
concurrent sessions:**

- Shape: **done** — `InteractiveSession` carries both users, audit ID, type,
  confidence, and match basis; `Match` refuses to attribute under ambiguity
  (tested).
- Substance: **weak** — confidence is hardcoded `"high"` for what the design
  target explicitly calls best-effort `namespace/pod/container` matching; the
  match basis is a free-form string, not a taxonomy; and only `FileAccess` is
  enriched (no process/network evidence, no session ID propagated on signals).

### 2. Prototype `giles/spo-uid-tracking` (commit `f9324ed3a2`, 13 files)

Inspected via `git show f9324ed3a2` only. Files: proto + generated changes for
the admission-control stream, a new Sensor-side identity cache, admission
manager and push wiring, `sensor/kubernetes/sensor/sensor.go`, and a
whitespace-only change to `proto/storage/file_access.proto`.

- `proto/internalapi/sensor/admission_control.proto`: new message
  `AdmissionControlIdentity{request_uid, username, namespace, pod, container}`.
- `proto/internalapi/sensor/admission_control_iservice.proto`:
  `MsgFromAdmissionControl` (previously empty) gains a oneof with
  `admission_identity`.
- `sensor/admission-control/manager/manager_impl.go`: after an **allowed**
  runtime admission request, converts the `admission.AdmissionRequest` via
  `kubernetes.AdmissionRequestToKubeEventObj` and pushes an
  `AdmissionControlIdentity` (RequestUid = `req.UID`, username, namespace,
  pod, container from `PodExecArgs`/`PodAttachArgs`) onto a buffered channel
  (capacity 100, drop-on-full with a warning log).
- `sensor/admission-control/settingswatch/sensor_push.go`: new
  `sendAdmissionIdentities` goroutine forwards identities over the existing
  admission-control `Communicate` gRPC stream to Sensor.
- `sensor/common/admissioncontroller/management_service.go`: Sensor-side
  `Communicate` records received identities into the cache (previously logged
  "not sure what to do with it").
- `sensor/common/admissionidentity/cache.go` (new): expirable LRU
  (`hashicorp/golang-lru/v2/expirable`), `maxEntries = 1024`,
  `retention = 5 * time.Minute`, keyed by request UID, with `Record`/`Lookup`.
- `sensor/kubernetes/sensor/sensor.go`: instantiates the cache and passes it
  to `admissioncontroller.NewManagementService`.
- `proto/storage/file_access.proto`: **only a blank line was added** — the
  intended FileAccess user field was never written.

**How SPO-sourced identity was meant to reach Sensor:** the ACS admission
webhook (which already intercepts `pods/exec`/`pods/attach`/`pods/portforward`
Connect operations on master — see question 4) captures the authenticated user
at admission time and keys it by the admission **request UID**. SPO's node
enrichment (the json-enricher in the demo, which annotates node-debug pods
with `nodedebuggingpod.spo.io/SPO_EXEC_REQUEST_UID`) attaches the same request
UID to process records; the intended join was: process/file evidence carrying
a request UID -> `admissionidentity.Cache.Lookup(requestUID)` -> user.

**Done:** admission-side capture (username, namespace, pod, container), gRPC
transport to Sensor, bounded expiring cache. **Missing:** nothing in the branch
ever calls `Lookup` — the cache is populated but unconsumed; there is no proto
field carrying a request UID (or session ID) on `ProcessSignal` or
`FileActivity`; the FileAccess field was never added; there is no SPO-side feed
producing request UIDs inside ACS. The join is an unimplemented intention. Two
further caveats: (1) the admission request UID and the kube audit ID are
different identifiers — `giles/k8s-fam-correlation` keys sessions by audit ID,
so a unified design must reconcile the key spaces; (2) the whole path depends
on ACS admission control being enabled and its webhook registered for
exec/attach Connect — clusters with admission control disabled capture nothing.

### 3. Master file-activity signal path end to end (worktree = master content)

1. **Collector/FACT -> Sensor transport**:
   `proto/internalapi/sensor/sfa_iservice.proto` —
   `service FileActivityService { rpc Communicate(stream FileActivity) }`.
   `proto/internalapi/sensor/sfa.proto` — `FileActivity{timestamp, process
   (ProcessSignal), oneof file{creation, unlink, rename, permission,
   ownership, open, write, xattr_set, xattr_remove, acl}, hostname}`. The
   `ProcessSignal` in `proto/internalapi/sensor/collector.proto` carries `id`,
   `container_id`, `creation_time`, `name`, `args`, `exec_file_path`, `pid`,
   `uid`/`gid`, `scraped`, `lineage_info` (`parent_uid`,
   `parent_exec_file_path`), and SFA fields `login_uid`, `username`,
   `in_root_mount_ns`. There is **no session or user-identity field beyond the
   Linux-level uid/login_uid**.
2. **Sensor gRPC service**: `sensor/common/filesystem/service/service_impl.go`
   — `CollectorOnly()` auth; `Communicate` -> `receiveMessages` ->
   `activityChan`.
3. **Sensor filesystem pipeline**:
   `sensor/common/filesystem/pipeline/pipeline.go` — `run()` pulls
   `activityChan`; `processFileActivity` builds a `ProcessIndicator` from the
   raw signal (`buildIndicator`); host processes (no container ID) bypass
   enrichment; container processes are buffered (100 per process, 10 000
   global, 5-minute TTL) and published unenriched on the internal pubsub
   (`UnenrichedProcessIndicatorTopic`) to be joined with the enriched
   indicator later, or enriched directly via
   `clusterEntities.LookupByContainerID` + `PopulateIndicatorFromContainer`
   (legacy path). `translateWithIndicator` produces the `storage.FileAccess`
   (paths, operation, metadata, process indicator, hostname, timestamp) and
   hands it to `detector.ProcessFileAccess`.
4. **Process lineage handling**: `sensor/common/processsignal/pipeline.go` /
   `enricher.go` — process signals from Collector's signal service are
   enriched with deployment/pod/container metadata from the
   `clusterentities.Store` (LRU of 100 000 unenriched indicators grouped per
   container, container metadata callback, 30-second container expiry);
   `PopulateIndicatorFromContainer` sets `DeploymentId`, `ContainerName`,
   `PodId`, `PodUid`, `Namespace`, `ContainerStartTime`, `ImageId`. Lineage is
   recorded as `(parent_uid, parent_exec_file_path)` only — there is **no
   mechanism today to inherit an arbitrary field (like a session ID) from
   parent to child processes**; any such tag must be copied by the producer
   (FACT) or re-derived by joining `(pid, pid start time)`.
5. **Storage proto**: `proto/storage/file_access.proto` — `FileAccess{file,
   operation, moved, timestamp, process (ProcessIndicator), hostname}`;
   embedded in alerts via `proto/storage/alert.proto` (`file_access = 8`,
   tagged `search:"-" sql:"-"`); explicitly has no primary key — it only
   exists inside `Alert.Violation`.
6. **Detection**: `sensor/common/detector/detector.go` — file-access queue or
   `DetectorFileAccessTopic` lane; `enrichFileAccess` snapshots the deployment
   or node; `detectAndAlertForFileAccess` calls
   `unifiedDetector.DetectNodeFileAccess` / `DetectFileAccessForDeployment`;
   matchers in `pkg/booleanpolicy/matcher.go` with
   `pkg/booleanpolicy/augmentedobjs/meta.go` (`FileAccessMeta` combines file
   access and process criteria); violation messages printed by
   `pkg/booleanpolicy/violationmessages/printer/file_access.go`.
7. **Central**: alert results flow over the Sensor message stream into
   `central/detection/alertmanager/alert_manager_impl.go` (merges new
   FileAccess violations into existing alerts, `mergeFileAccessViolations`);
   GraphQL exposure in `central/graphql/resolvers/generated.go` (`FileAccess`
   type on `Alert.Violation`); notifier integrations in
   `central/splunk/violations.go` (`extractFileAccessInfo`).
8. **UI**: `ui/apps/platform/src/types/fileAccess.proto.ts` and
   `ui/apps/platform/src/Containers/Violations/Details/FileAccessCardContent.tsx`
   (file operation, time, hostname, actual/effective path, process name/UID,
   file metadata owner/group/UID/GID).

**Every touchpoint where a sessionID/user field would need to flow:**

- **Protos** (each requires `make proto-generated-srcs` and UI type
  regeneration): `proto/internalapi/sensor/collector.proto` (`ProcessSignal`
  + session ID and attribution type), `proto/internalapi/sensor/sfa.proto`
  (inherits via `process`), `proto/storage/process_indicator.proto`
  (`ProcessIndicator`/`ProcessSignal` carry the ID through enrichment),
  `proto/storage/file_access.proto` (identity/evidence block on `FileAccess`),
  optionally `proto/storage/kube_event.proto` (user fields exist; the
  prototype adds `audit_stage`). `proto/storage/alert.proto` needs no change
  (embedding).
- **Collector/FACT (out of repo)**: tag the root exec process and copy the
  session ID through fork/clone/exec lineage; buffer for the
  binding-notification race.
- **Sensor**: `sensor/common/filesystem/service/service_impl.go` (transparent,
  no change if the proto carries the field); `sensor/common/filesystem/
  pipeline/pipeline.go` — `buildIndicator` (copy session ID from the signal),
  `translateWithIndicator` (resolve session ID -> identity and materialise
  into `FileAccess`), match timing interacts with the
  `containerID:processSignalID` buffering; `sensor/common/processsignal/`
  (lineage/session resolution if signals arrive untagged); a new
  session-binding consumer component (see question 5) plus
  `sensor/common/pubsub/topic.go` if a new lane/topic is introduced;
  `sensor/common/clusterentities/store.go` if container-ID joins are needed;
  `sensor/common/detector/detector.go` (`enrichFileAccess` snapshot if
  identity is resolved at detection time).
- **Central**: `pkg/booleanpolicy/augmentedobjs/` + `matcher.go` if policies
  gain session-user criteria (the audit prototype deliberately avoids this);
  `pkg/booleanpolicy/violationmessages/printer/file_access.go` (violation
  text); `central/detection/alertmanager/alert_manager_impl.go` (merge
  semantics must not conflate differently-attributed violations);
  `central/graphql/resolvers/generated.go` + schema; `central/splunk/
  violations.go` (notifier fields).
- **UI**: `ui/apps/platform/src/types/fileAccess.proto.ts`,
  `ui/apps/platform/src/Containers/Violations/Details/FileAccessCardContent.tsx`,
  optionally search/filter surfaces (driven by `@gotags: search:` on the new
  proto fields).

### 4. Existing user-identity precedents on master

- **Kube event user retention**: `pkg/kubernetes/event.go` —
  `AdmissionRequestToKubeEventObj` maps `PodExecOptions`/`PodAttachOptions`/
  `PodPortForwardOptions` admission requests (operations `Connect`/`Create`)
  into `storage.KubernetesEvent`, retaining
  `User{username, groups}` from `req.UserInfo`. `proto/storage/kube_event.proto`
  tags these with `policy:"Kubernetes User Name"` /
  `policy:"Kubernetes User Groups"`, so policies can already key on the
  exec'ing user.
- **Sensor admission-control service**: `sensor/admission-control/manager/
  evaluate_runtime.go` — the ValidatingWebhookConfiguration intercepts
  exec/attach/portforward Connect requests; `shouldBypassRuntimeDetection`
  consults per-cluster bypass lists of users, groups, and system service
  accounts; `evaluatePodEvent` runs k8s-event policies with the user-bearing
  event. This is the hook the SPO-UID prototype extends.
- **Deployment-creation attribution**: master has **no** durable
  per-deployment creator field. The closest signals are the admission-time
  user above, audit-log CREATE events (user in `user.username`), and
  `pkg/kubernetes/annotations.go`, which only *trims* annotations like
  `kubectl.kubernetes.io/last-applied-configuration` (annotation hygiene, not
  attribution). The design target's premise that labels/annotations are not
  forensic evidence is reflected here.
- **Compliance audit ingestion**: `compliance/collection/auditlog/
  auditlog_impl.go` (node-side tail of the audit log; master allow-lists
  `ResponseComplete`/`Panic` and non-interactive resources) and
  `auditevent.go` `ToKubernetesEvent` (retains `User`,
  `ImpersonatedUser`, `SourceIps`, `UserAgent`, `RequestUri`,
  `ResponseStatus`; audit ID as event ID). Flow: node compliance service ->
  `sensor/common/compliance/service_impl.go` `Communicate` -> internal pubsub
  `DetectorAuditLogTopic` -> `sensor/common/detector/detector.go`
  `runAuditLogEventDetector` -> `unifiedDetector.DetectAuditLogEvents`
  (`sensor/common/detector/unified/detector_impl.go`) -> alerts. Restart
  state is managed by `sensor/common/compliance/auditlog_manager.go`, which
  persists audit-log file offsets to Central and re-syncs on reconnection.

### 5. Join-point analysis for a verified session-binding feed

The design target has Sensor receive two inputs: the OpenShift
`PrivilegedSession` record (session ID -> user and request evidence) and
FACT/Collector events tagged with the opaque session ID. Candidate
consumption points, contrasted with the audit prototype:

- **A. Filesystem pipeline at translation time** (what
  `giles/k8s-fam-correlation` implements): join by
  `(namespace, pod, containerName)` against active audit sessions. Simple and
  already prototyped, but only best-effort: name-keyed, not ID-keyed, and
  ambiguous under concurrency.
- **B. Process-signal enrichment**: if FACT tags signals with a session ID
  (or `(host PID, PID start time)` for lookup), the natural insertion points
  are `sensor/common/processsignal/enricher.go` (resolve the session ID to an
  identity alongside container-metadata enrichment, reusing the
  LRU-buffer-then-enrich pattern) and/or a new internal pubsub lane
  (`sensor/common/pubsub/topic.go`). The file pipeline's buffered-activity
  mechanism (hold events per `containerID:processSignalID` until the enriched
  indicator arrives) is exactly the Sensor-side race buffer the design target
  asks FACT to have; a verified binding arriving late (~185 ms in the lab)
  would be absorbed the same way.
- **C. New Sensor gRPC service**: a node-local Session Attribution Agent could
  stream `ProcessSessionBinding`s to a dedicated service analogous to
  `sensor/common/filesystem/service/service_impl.go` (CollectorOnly-style
  auth, Unix-socket or mTLS per the design target).
- **D. PrivilegedSession watch**: Sensor's orchestrator watch machinery
  (`sensor/common/orchestrator`, `sensor/kubernetes/listener`) could materialise
  session ID -> user/request evidence from the CRD, combined with either feed.

**Where the audit prototype's exactly-one-active-session rule bites:**
two overlapping execs into one container (no attribution — silently);
overlapping exec + attach (same); missed `ResponseComplete`/`Panic` (audit
tail gap, Sensor offline mode, or Sensor restart) leaving a stale session
until TTL expiry, during which a *single* stale session can be wrongly
attributed; pod-name reuse within the TTL (stale match against a new pod);
audit `requestURI` without a `container` query parameter (container key empty,
session skipped). The design target's demand — never infer the user merely
because a session was active near the event — is only partially honoured: the
prototype does refuse ambiguous matches, but it cannot distinguish "one
verified binding" from "one surviving best-effort session".

**What a matchBasis taxonomy (`direct-exec-root` /
`active-session-best-effort` / `unattributed`) would require:**

- A closed enum on the proto (the prototype's free-form strings cannot be
  validated or evolved safely).
- The verified path: session ID present on the process signal (from a signed
  runtime PID-lineage binding), resolved to identity from the
  PrivilegedSession/binding feed; only this may claim high confidence and name
  the user in user-facing text.
- The best-effort path: exactly-one-active-session name-keyed match, always
  labelled as best-effort (the prototype's "high" confidence must be
  downgraded here).
- Unattributed: no session, ambiguous sessions, or missing binding — retained,
  never inferred.
- Downstream discipline: policy criteria and UI labels distinguishing the
  classes; alert merging (`central/detection/alertmanager`) must not coalesce
  violations with different matchBasis or users; missing-attribution
  observability so gaps are visible (design target's bypass-prevention
  requirement).

### 6. Gap list vs design target

1. **PrivilegedSession record consumption** — nothing on master or either
   prototype maps session ID -> user from a platform-owned durable record
   (the OpenShift CRD itself does not exist yet).
2. **Session ID on runtime events** — no session field on `ProcessSignal`,
   `FileActivity`, `ProcessIndicator`, or `FileAccess` on master or
   `giles/spo-uid-tracking` (blank-line diff only);
   `giles/k8s-fam-correlation` adds identity fields but **no session ID**,
   so historical evidence cannot be re-joined after Pod deletion.
3. **Identity snapshot completeness** — prototype snapshots only usernames;
   the design target's evidence set (groups, request type, target identity,
   timestamps) is absent despite the audit event carrying source IPs, user
   agent, and groups.
4. **Verified-binding claim discipline** — prototype hardcodes
   `confidence: "high"` with `matchBasis: "namespace/pod/container"`,
   contradicting the design target's requirement to distinguish best-effort
   matches from verified runtime process bindings.
5. **Evidence scope** — only `FileAccess` is enriched; the design target wants
   the same context on process, network, and other runtime-policy evidence.
6. **Attach semantics** — the design target says attach must be excluded from
   exact attribution or reported as active-but-unattributed; the prototype
   attributes attach identically to exec with "high" confidence.
7. **Missing-attribution observability** — no metrics on tracker state,
   matches, drops, or expiry in either prototype.
8. **Bounded/immutable keying** — no container-ID/Pod-UID resolution; matches
   ride on mutable names (see question 1).
9. **Restart/persistence semantics** — both prototype caches (session tracker,
   admission-identity LRU) are in-memory with no re-sync path.
10. **Neither prototype is landed** — single unmerged commits on personal
    branches; generated code is committed but regeneration reproducibility
    was not verified; no feature flag gates the enrichment
    (`giles/k8s-fam-correlation` enriches whenever the tracker is wired).

### 7. Constraints

- **Sensor restart semantics for in-memory session state**: on restart the
  tracker/cache is empty; the audit-log reader resumes from the offset
  persisted via `sensor/common/compliance/auditlog_manager.go` (state synced
  with Central on reconnection), so pre-restart `ResponseStarted` events are
  not replayed and cannot rebuild live sessions. Any production design needs
  either persisted session snapshots, a bounded audit re-read window, or an
  explicitly accepted enrichment gap after restart.
- **Protobuf regeneration**: proto changes require `make
  proto-generated-srcs` (Go pb/vtproto, swagger, GraphQL-generated types,
  `ui/apps/platform/src/types/*.proto.ts`). The prototypes commit generated
  artefacts (e.g. `file_access_vtproto.pb.go` +1269 lines in
  `giles/k8s-fam-correlation`) — these must be regenerated, not hand-carried.
- **Feature flags**: file activity is gated by `SensitiveFileActivity`
  (`ROX_SENSITIVE_FILE_ACTIVITY`, default-enabled, `pkg/features/list.go`)
  and the internal pubsub by `SensorInternalPubSub` (`ROX_SENSOR_PUBSUB`).
  Session enrichment needs its own flag; FACT-facing configuration flows
  through `sensor/common/filesystem/settings_manager.go` (`fact-config`
  ConfigMap).
- **UI surface**: the violation detail card is the only display point today;
  new fields need TS types + card items, and searchability requires
  `@gotags: search:` on the proto (none set by the prototype; note
  `FileAccess` is embedded in alerts with `search:"-"`, so no DB migration is
  needed unless search tags are added — schema-only additions are handled by
  GORM auto-migrate per repo AGENTS.md).
- **Admission-control dependency** (SPO path): the webhook must be enabled and
  registered for exec/attach Connect subresources; clusters with admission
  control disabled get nothing. The audit-log path instead requires node
  compliance scanning plus the audit log path to be configured.

### 8. Prior demo and lab evidence

`/home/ghutton/infra/demos/user-correlation/` is a Splunk-based correlation
demo: OpenShift API audit records and SPO json-enricher process records
(carrying the admission request UID via the SPO node-debugging webhook's
`SPO_EXEC_REQUEST_UID` annotation) are forwarded to one Splunk index through
Red Hat OpenShift Logging. `SEARCHES.md` sessionises `oc debug node` sessions
by the `attach` audit `ResponseStarted`/`ResponseComplete` stages and joins
ACS file-access violations by namespace/pod; a process-linked search joins
SPO request UID -> audit user -> RHACS violation by host, PID, and time
(+-2 s); SSH sessions are derived from auditd `USER_START`/`USER_END`. It
demonstrates that session-level user correlation is feasible with existing
signals, while stating plainly that the attribution is session-level (not
proof for an individual file modification) and that the SPO request UID is
best-effort, workload-forgeable metadata.

`/home/ghutton/infra/useful/exec-audit-lab-artifacts/run-20261023T141033Z/`
(`report.md`, `bindings.json`, `agent.log`, `target-files.txt`) is a
node-agent lab run: the simulated session `23646cd6...` was bound to root PID
509195 (start ticks 10421513) and descendant 509197 in container `d2cca4...`;
the controlled writes landed at 14:10:48.524/526 and the binding was observed
at 14:10:48.711 — about 185 ms after the file writes — and `/proc` polling
missed a short-lived nested shell. This validates PID-lineage propagation of a
session marker and quantifies the binding lag a consumer must buffer through,
while the report itself flags that the marker is visible and forgeable and
that FACT/Collector integration is the next phase.

## Gap list vs design target

See question 6 above for the enumerated list; in short, master and both
prototypes cover only the best-effort correlation tier. Everything the design
target requires for verified attribution — the platform-owned
PrivilegedSession record, the signed session capability and node agent, the
session ID on runtime events, Sensor-side resolution of session ID to
identity before policy evaluation, and the matchBasis/confidence taxonomy
with unattributed retention — remains to be built, and the OpenShift-side
components are prerequisites owned outside ACS.

## Open questions

- Which identifier unifies the key spaces: the kube audit ID (used by
  `giles/k8s-fam-correlation`), the admission request UID (used by
  `giles/spo-uid-tracking` and SPO), or a new opaque PrivilegedSession
  session ID? The three are distinct today.
- Should Sensor resolve session IDs at process-signal enrichment time
  (pipeline B) or at FileAccess translation time (pipeline A), given the
  ~185 ms binding lag and the existing per-process buffering? Both need a
  bounded staleness window.
- Can the exactly-one-active-session rule be preserved for best-effort
  attribution once a verified binding feed exists, or should best-effort
  attribution be retired entirely to avoid two competing confidence claims?
- What is the Sensor restart story: persisted session snapshots in Central, a
  bounded audit re-read window, or an accepted gap with explicit
  unattributed labelling?
- Does policy matching on the interactive-session user belong in
  `booleanpolicy` (the audit prototype deliberately defers this with
  "field will not be used for policy detection" on `audit_stage`)?
- Is the SPO request-UID feed trustworthy enough for anything beyond
  best-effort, given the demo's own assessment that a workload can alter it?
- What UI/search surface (if any) should the identity snapshot get, given
  `FileAccess` is embedded in alerts with search disabled?
