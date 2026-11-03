# Synthesis: Session-User Correlation for File Activity

Controller-authored synthesis for TaskWarrior task `a27a78e8-44c6-46d7-b78b-484d11a6b0fa`.
Inputs: `spo-extension-options.md` (Stream A), `openshift-correlation-options.md`
(Stream B), `acs-integration-surface.md` (Stream C), the design target
`PRIVILEGED_SESSION_ATTRIBUTION_RESEARCH.md` (repo root, untracked), the
exec-audit-lab artifacts (Oct 2026), and the two ACS prototype branches.

## Summary

No shipped OpenShift mechanism carries the API-server identity of an `oc exec`
request to the node (Stream B §5): CRI has no user field, kubelet authorises only
the apiserver, and CRI-O tracks exec PIDs in memory. Meanwhile the exact design we
parked in Oct 2026 — mutate the exec command so a request identifier lands in the
process environment — shipped upstream in the Security Profiles Operator
(`execmetadata.spo.io` webhook, PR #2878, Jun 2025) as part of Red Hat's OCP
user-activity-auditing programme (OCPSTRAT-846, OCP 4.20 target). SPO's own
documentation classifies the result as forgeable, "best effort and not audit
evidence". The recommendation is therefore a two-tier attribution model: consume
the shipped best-effort joins now under honest labels, and pursue the trusted tier
(signing, CRI-O reporting, platform-owned session records) through the existing
OCPSTRAT-846 programme and SPO upstream, with SPO as the natural node-agent
delivery vehicle. All viable near-term marker sources are presented below with
trade-offs, per review decision.

## The attribution model

Attribution claims must be keyed on a closed `matchBasis` enum, not free-form
confidence strings. The audit prototype's hardcoded `confidence: "high"` for a
name-keyed timing join (Stream C §1) is the defect this model fixes.

| matchBasis | Evidence | May name the user? | Forgeable in-container? | Status |
|---|---|---|---|---|
| `verified-pid-lineage` | Signed session capability → CRI-O binding → (host PID, start time) → lineage | Yes, forensic claim | No — capability never enters the container | Future, platform-gated (design target) |
| `request-uid-join` | `SPO_EXEC_REQUEST_UID` marker captured on the exec root at process start, inherited by descendants; joined to the unforgeable apiserver-side record | Yes, with explicit best-effort caveat | Yes — env readable/forgeable/strippable; cross-session framing possible (Stream B §3) | Buildable now |
| `active-session` | Exactly one active exec/attach session for (namespace, pod, container) at event time, from audit lifecycle | Display only, lowest tier; refuse under ambiguity | The join is inference, not proof | Buildable now (prototype exists) |
| `unattributed` | No session, ambiguous sessions, or missing binding | No — retained, never inferred | n/a | Now |

Rules:

1. Only `verified-pid-lineage` may carry high confidence in user-facing text.
2. `request-uid-join` and `active-session` are always labelled best-effort; the
   apiserver-side half of the request-UID record (audit event + webhook
   annotation) is trustworthy, only the in-container copy is soft.
3. Alert merging (`central/detection/alertmanager`) must not coalesce violations
   with different matchBasis or different attributed users.
4. Missing attribution must be observable (metrics on tracker state, drops,
   expiry, marker-stripping anomalies) — the design target's requirement that
   gaps never silently become successful attributions.

## Marker sources for the request-UID join — all options

The join needs two halves: (a) a per-process marker reaching ACS telemetry, and
(b) a `requestUID → user` map (next section). Options for (a):

### Option 1 — FACT/Collector captures `SPO_EXEC_REQUEST_UID` at exec

Revives the parked task-18 design, now de-risked by upstream precedent.

- Mechanics: SPO `execmetadata.spo.io` rewrites the exec command to run through
  `env(1)` with `SPO_EXEC_REQUEST_UID=<AdmissionRequest.UID>`; FACT captures
  that single variable at `sched_process_exec` via a BPF tracepoint (exactly
  what SPO's own `bpfprocesscache` does — env capture bounded to 4096/8192
  bytes, only the request UID retained; Stream A §4) and propagates it through
  process lineage; the marker flows in-band on `ProcessSignal` → `FileAccess`
  through the existing pipeline (Stream C §3).
- Prerequisites: SPO deployed with `enableExecMetadata`; collector-repo work
  (env capture + lineage propagation); Sensor-side map consumer; privacy review
  (settled upstream to an opaque UID, no username — Stream A §6).
- Strengths: PID-precise; marker present at process start (no join race);
  reuses the existing signal path with no new ingestion surface; the BPF
  tracepoint closes the lab's short-lived-process miss.
- Weaknesses: collector-repo changes; exec only (attach uncovered, consistent
  with the design target); clusters without SPO get nothing from this tier.
- Effort: medium. ACS touchpoints are enumerated in Stream C §3 (protos:
  `collector.proto`, `sfa.proto`, `process_indicator.proto`,
  `file_access.proto`; pipeline: `buildIndicator`, `translateWithIndicator`).

### Option 2 — Sensor consumes the SPO JSON enricher output

- Mechanics: SPO's JSON enricher (auditd reader with the BPF process cache as
  fallback) emits per-process JSON lines carrying `requestUID`; a Sensor-side
  consumer ingests them (node-local log tail or the daemon's gRPC enricher
  socket — Stream A open question 7) and joins to file events by host PID and
  time.
- Prerequisites: SPO with `enableJsonEnricher`; a new Sensor ingestion path.
- Strengths: no collector changes; SPO's BPF cache already solves
  short-lived-process capture at the source.
- Weaknesses: the log format is `spo/v1_alpha` and its e2e hardening is still
  in progress (OCPNODE-4767 / SPO PR #3496, open Nov 2026); a second telemetry
  path parallel to the ACS signal pipeline; the join on our side is by
  (host PID, time) rather than an in-band marker.
- Effort: medium (new consumer + join logic; no FACT involvement).

### Option 3 — Audit-session tracker (no marker; session-level)

Evolves `giles/k8s-fam-correlation` (Stream C §1).

- Mechanics: Sensor ingests exec/attach audit lifecycle
  (`ResponseStarted`/`ResponseComplete`/`Panic`), maintains bounded sessions
  keyed by audit ID, and enriches `FileAccess` when exactly one session is
  active for the key.
- Prerequisites: audit-log path configured (compliance collection); no SPO, no
  collector work.
- Strengths: already prototyped (bounded 1024/5-min TTL, ambiguity refusal
  tested); covers `attach` as an active-but-weak tier; no external dependency.
- Weaknesses: session-level, not process-level; name-keyed today (pod-name
  reuse, missing container parameter); audit delivery is seconds-class, so
  `ResponseStarted` can arrive after early file writes — the lab's ~185 ms
  binding lag at cluster scale; Sensor restart loses live sessions (Stream C §7).
- Hardening needed: container-ID/Pod-UID resolution, metrics, restart story,
  and the labelling fix (this tier is `active-session`, never "high").
- Effort: low–medium.

### Comparison and phasing

| | Option 1 (FACT env capture) | Option 2 (SPO enricher consumption) | Option 3 (audit tracker) |
|---|---|---|---|
| Evidence precision | Process + lineage | Process (joined externally) | Session only |
| New ACS-repo work | Proto fields + pipeline | Ingestion consumer + join | Harden prototype |
| New collector-repo work | Yes | No | No |
| SPO deployment required | Yes (`enableExecMetadata`) | Yes (`enableJsonEnricher`) | No |
| Depends on log format stability | No | Yes (alpha) | No (audit schema stable) |
| Covers attach | No | No (SPO does not intercept `pods/attach`) | Yes (weak tier) |
| Latency sensitivity | None (marker at start) | Log-path latency | Audit-path latency (race) |

Suggested phasing (proposal, not a single pick — all options remain on the
table per review):

1. **Phase 1** — land the taxonomy and labelling fix, plus the hardened audit
   tracker as the `active-session` tier: no external dependency, exists today.
2. **Phase 1b** — land the `requestUID → user` map in Sensor (next section);
   it is the common prerequisite for Options 1 and 2.
3. **Phase 2** — Option 1 as the primary `request-uid-join` tier (strongest
   near-term evidence); Option 2 as the fallback where collector updates are
   not possible.
4. **Phase 3** — the verified tier via platform engagement (below).

## Building the requestUID → user map

Three routes, all resolving the same key; standardise on the
**AdmissionReview UID** — it is generated by the apiserver per request and
shared by every webhook in the chain, so SPO's mutating webhook, ACS's
admission-control webhook (which already intercepts exec/attach Connect with
full `userInfo`), and the audit event's `execmetadata.spo.io/SPO_EXEC_REQUEST_UID`
annotation all carry the same identifier. The kube `auditID` is a *different*
identifier and remains the session-lifecycle key for Option 3; do not conflate
them (Stream C §2).

1. **ACS admission webhook** (evolves `giles/spo-uid-tracking`): the
   admission-control service already sees exec/attach Connect requests with
   user and `req.UID`; the prototype built the gRPC push and the bounded
   expiring LRU — the missing piece is a consumer and a proto field carrying
   the UID (Stream C §2). Requires ACS admission control enabled.
2. **Audit annotation join**: audit events for exec carry the SPO-stamped
   request-UID annotation and the user at the `Default` OCP profile (Stream B
   §1); the audit ingestion path can build the same map without the admission
   webhook. Requires the audit-log path and SPO's webhook.
3. **Platform feed** (future): the `PrivilegedSession` record or SPO binding
   agent supplies the map directly.

OCP corroborating evidence available in audit events regardless of route:
`user.extra[authentication.kubernetes.io/credential-id]`, the OAuth
login/logout trail, and `sourceIPs` (Stream B §1).

## Upstream and platform engagement

### SPO (kubernetes-sigs, SIG-Node)

- Maintainers: `ccojocar`, `saschagrunert`, `JAORMX`, `jhrozek`; reviewers
  `Vincent056`, `ngopalak-redhat` — the request-UID/JSON-enricher line is
  authored by `ngopalak-redhat`; Red Hat involvement is strong and the OCP
  delivery is tracked with OCPSTRAT/OCPNODE/OCPBUGS references (Stream A §7).
- Propose upstream: (a) a stable `requestUID` contract (the JSON enricher
  format is `spo/v1_alpha`); (b) discussion of a node-local session-binding
  agent container in spod, consuming a platform-provided trust anchor — SPO
  already has the (PID, start-time) keying, PID-reuse defences, and
  node-local gRPC socket patterns (Stream A §5); (c) the daemon's enricher
  gRPC socket as a consumer interface for node telemetry agents.
- SPO cannot issue forensic-grade trust: signing, CRI-O reporting, and a
  platform-owned session record are control-plane surfaces with no SPO
  interface, and OCP's supported audit configuration cannot feed an in-cluster
  webhook sink (Stream A §5). Frame SPO as the delivery vehicle, not the
  issuer.

### OCPSTRAT-846 / OCPNODE-3426 (user activity auditing, OCP 4.20)

The existing Red Hat programme that landed the SPO request-UID work. The
trusted tier — `PrivilegedSession` record, signed session capability,
kubelet/CRI-O propagation, and a node-local attribution agent — should be
proposed into this programme rather than as a fresh initiative; the design
target document is the input. Also track `oc debug` labelling
(openshift/oc PR #2055) which the SPO node-debug webhook relies on.

### CRI-O / Kubernetes

- CRI-O: no exec-session reporting interface exists today (exec PIDs tracked
  in memory only; NRI covers pod/container lifecycle, not exec — Stream B §5).
  The design target's trusted propagation needs a cri-o extension; open the
  conversation early.
- Kubernetes: WebSocket-based exec is authorised as `get`, not `create`
  (`AuthorizePodWebsocketUpgradeCreatePermission` gate) — any bypass
  prevention or audit-based detection must handle both verbs (Stream B §4).

## Bypass prevention

A validating webhook on `CONNECT` for `pods/exec` can require mediation
fail-closed; RBAC must gate both `create` and `get` on `pods/exec` (and
`pods/ephemeralcontainers` updates for debug flows). Holders of `nodes/proxy`,
node SSH, and CRI-socket access bypass admission entirely — a platform-level
concern the engagement section must carry, not an ACS control (Stream B §4).

## Recommended next steps

1. **Taxonomy + labelling** (ACS, small): closed `matchBasis` enum on
   `FileAccess.InteractiveSession`; downgrade the hardcoded confidence; alert
   merge discipline. Evolves `giles/k8s-fam-correlation`.
2. **requestUID map** (ACS, small–medium): wire the `admissionidentity` cache
   consumer and the proto field carrying the request UID. Evolves
   `giles/spo-uid-tracking`.
3. **Collector design doc** (collector repo): `SPO_EXEC_REQUEST_UID` capture at
   `sched_process_exec` with lineage propagation — revive the parked task-18
   design, citing the upstream privacy precedent (opaque UID only).
4. **Lab extension**: deploy SPO on the existing OCP lab cluster with
   `enableExecMetadata` + JSON enricher; measure end-to-end requestUID flow and
   answer Stream A open question 4 (does the tracepoint capture close the
   ~185 ms binding lag; container-ID resolution still rides the pod watch).
5. **Upstream conversations**: SPO maintainers on the binding-agent/stable
   contract; OCPSTRAT-846 for the trusted tier; cri-o on exec reporting.

## Open questions (consolidated)

- Key-space unification beyond request UID vs auditID (Stream C §5): does the
  future `PrivilegedSession` session ID subsume both, and what migrates?
- Sensor restart story for session state (persisted snapshots vs bounded
  audit re-read vs accepted gap) — Stream C §7.
- Does best-effort attribution retire once verified bindings exist, or do the
  tiers coexist? (Stream C open questions.)
- Marker-stripping: should unattributed activity during an active session be
  treated as suspicious rather than merely unattributed (Stream B open
  questions)?
- Product surface: should matchBasis/user fields become searchable
  (`@gotags: search:`), given `FileAccess` is alert-embedded with search
  disabled today (Stream C §7)?
- SPO GA state: which OCP/SPO versions ship `enableJsonEnricher` and the
  oc-debug label integration (Stream A open questions 1, 6)?
