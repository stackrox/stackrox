# Security event plan

This document consolidates the original issue definition and its detailed
implementation and validation plan.

## Issue: External PolicyReport ingestion and vulnerability enrichment

### Problem

Customers use policy engines such as Kyverno, Gatekeeper, and Kubernetes
ValidatingAdmissionPolicy alongside ACS. Their findings are reported independently,
leaving customers without a unified, vulnerability-aware view of policy violations.

ACS already has the cluster context, deployment inventory, vulnerability data, and
multi-cluster visibility needed to make these external findings more actionable.

### Desired outcome

Ingest PolicyReport and ClusterPolicyReport resources through Sensor, persist them in
Central, expose them through a scoped read API, and enrich violations with relevant
deployment vulnerability context.

### Proposed scope

- Watch supported PolicyReport resources when their CRDs are available.
- Normalize external results into a stable ACS representation.
- Send create, update, and delete events from Sensor to Central.
- Reconcile current state after Sensor reconnects.
- Store violations in PostgreSQL with cluster- and namespace-aware access control.
- Provide a read-only API supporting filters such as cluster, namespace, source, policy, and severity.
- Correlate affected Kubernetes objects with ACS deployments.
- Attach a vulnerability summary, including critical and fixable CVE counts.
- Protect the integration with a feature flag during development and validation.

### Non-goals

- Deploying or managing external policies.
- Replacing the ACS policy engine.
- Performing admission control or enforcement.
- Automatically remediating violations.
- Building a complete UI experience in the initial milestone.

### Suggested milestones

1. Parse PolicyReport resources and cover normalization with unit tests.
2. Demonstrate PolicyReport events flowing from Sensor to Central.
3. Add reconciliation-correct storage and a scoped read API.
4. Correlate violations with ACS deployments.
5. Add vulnerability enrichment and validate at realistic volume.

### Key design questions

- What fields form a stable violation identity across updates and API versions?
- How are deleted or stale findings removed reliably?
- Which PolicyReport and OpenReports API versions should be supported?
- How should Pod findings map to higher-level ACS deployment objects?
- How should cluster-scoped findings interact with scoped access control?
- What event-volume and memory bounds are required for large clusters?

### Acceptance criteria

- A supported external policy engine can produce a violation that becomes queryable through ACS.
- Updates and deletions converge correctly, including after Sensor reconnects.
- Users cannot read violations outside their authorized cluster and namespace scopes.
- A violation associated with a known deployment includes a vulnerability summary.
- Clusters without the relevant CRDs experience no errors or material resource overhead.
- Existing ACS policy evaluation and enforcement behavior is unchanged.

### Risks

- PolicyReport APIs are evolving and may require multi-version compatibility.
- Incorrect object correlation could produce misleading enrichment.
- High-volume reports could increase Sensor or Central load.
- Treating reports as event history instead of current state could retain stale violations.

### References

- `projects/policy-engine-evolution/spec.md`
- `pkg/policyreport/`
- `sensor/kubernetes/listener/resources/policyreport/`
- `sensor/kubernetes/listener/watcher/policyreport/`
- `proto/internalapi/central/external_policy_violation.proto`
- `proto/storage/external_policy_violation.proto`

---

## Implementation and validation plan: Kyverno policy results as ACS violations

## Objective

When Kyverno reports a failing policy result for a Pod owned by a workload, ACS displays it as an ordinary alert on the owning deployment in the main Violations view.

The first production path is deliberately narrow:

```text
Kyverno PolicyReport/OpenReports PolicyResult
  -> Sensor SecurityEvent
  -> Pod-to-deployment resolution
  -> ACS SECURITY_EVENT policy evaluation
  -> ordinary AlertResults
  -> ordinary storage.Alert
  -> main Violations view
```

Implementation and validation advance together. Each phase must leave a demonstrable, testable artifact and has a checkpoint at which scope or design can be adjusted before the next layer is added.

## User-visible definition of done

Given a secured cluster running Kyverno:

1. A Kyverno audit policy reports a property of a Pod template, such as use of a privileged container, as noncompliant.
2. A Deployment creates a Pod that produces a failing Kyverno report result.
3. Sensor discovers the result and associates the Pod with its owning ACS deployment.
4. Central receives and stores an ordinary ACS alert.
5. The main Violations view shows the alert against that Deployment.
6. The alert identifies Kyverno, the reported policy and rule, the original Pod, result, reported severity, and message.
7. Existing cluster, namespace, deployment, policy, state, and severity filtering works.
8. Applicable ACS scopes and exclusions prevent unwanted alerts.
9. Fixing or deleting the violating workload causes the alert to resolve.
10. Sensor reconnect and report reordering do not create duplicates or leave stale active alerts.

## Guiding decisions

### Reuse the standard alert path

The primary user-facing object is `storage.Alert`. A separate integration-specific table or screen is not the MVP. Producer fields are retained as alert violation details.

### Treat reports as current state

PolicyReport resources are state aggregates, not durable event logs. Correctness is defined by idempotence and eventual convergence rather than global event ordering.

### Use a source-neutral event envelope

`SecurityEvent` is the canonical ingestion envelope. It contains common provenance, subject, resolved entity, and observation time plus a typed payload. The first payload is `PolicyResult`; later adapters may add `FileIntegrityResult` or `SecurityProfileDenial` without flattening their semantics into policy-result fields.

The producer boundary is represented by `SecurityEvent.source`, not by calling the event external. Kyverno, FIO, SPO, Compliance Operator, Kubernetes admission, and ACS are all potential security-event sources from a customer's perspective.

### Evaluate through an explicit event source

Add `SECURITY_EVENT` to the policy event-source model. Security-event criteria are valid only for that source, and payload-specific criteria are valid only for compatible event types. A default catch-all ACS policy makes initial Kyverno failures visible without requiring customers to author a policy first.

Validated against the current code (`pkg/booleanpolicy/validate.go`): the minimum-required-fields mechanism (`eventSourceRequirements`) is a plain `map[storage.EventSource]set.StringSet`, checked generically with no per-source logic. `DEPLOYMENT_EVENT`/`NOT_APPLICABLE` today require zero fields; `NODE_EVENT` requires exactly one (`File Path`). Adding `SECURITY_EVENT` with zero or one required field is a single map entry — no changes needed to the validation functions themselves. The new enum value does, however, need an explicit case added at several other non-exhaustive `switch policy.GetEventSource()` call sites for it to actually function end to end: `pkg/detection/compiled_policy.go` (compilation/matcher wiring, two switches), `pkg/detection/runtime/utils.go`, `central/policy/service/validator.go` (`validateEventSource` and its `isAuditLogEventSource`/`isNodeEventSource`-style helpers), and Sensor's settings-propagation managers (`sensor/common/filesystem/settings_manager.go`, `sensor/common/admissioncontroller/settings_manager_impl.go`). None of this is hard, but it's not isolated to one file — track it as a checklist when Phase 4 lands.

### Ship a default, built-in policy before a customer-facing criteria language

The MVP does not need customers to author their own `SECURITY_EVENT` match criteria. Split Phase 4 into two parts:

1. **MVP**: one feature-flagged, built-in `storage.Policy` row (visible/clonable in the Policies UI like any other default policy, but with a single trivial required criterion, e.g. "Security Event Type equals PolicyResult") that the detector matches against unconditionally for every actionable, deployment-resolved finding. Because it's a real `Policy`, exclusions, scoping/RBAC, enable/disable, notifier dispatch, and the standard `Alert` lifecycle/UI all work for free — none of that infrastructure is specific to how rich the match criteria are.
2. **Deferred**: the full `PolicyResult` field vocabulary (Reported Policy, Reported Rule, Policy Result, Reported Severity as independently matchable criteria) and the field-metadata/validation/matcher build-out this doc originally scoped as all of Phase 4. Build this only once there's real demand for customer-authored filtering (e.g. "only alert on HIGH severity Kyverno findings") — retrofitting it onto the default-policy plumbing later is additive, not a rewrite, since both share the same `SECURITY_EVENT` source and detector path.

This was chosen over building a separate, non-policy-engine violation store (what the deleted prototype under `central/externalpolicyviolation/` attempted): a parallel system would have to reimplement exclusions/scoping/notifiers/lifecycle/UI rendering from scratch, and would fragment the user's mental model of "where do I check for problems" — something the Deferred scope section below already rules out ("a separate security-events or integration-findings UI").

### Keep signal and violation semantics separate

A `SecurityEvent` is reported evidence. An ACS policy decides whether that evidence creates a violation. Producer severity remains reported data; ACS policy severity controls the resulting alert unless a later product decision explicitly introduces severity inheritance.

### Reuse ResourceAction for lifecycle

The report result is current state. `ResourceAction` expresses observation, update, and removal during transport. Stable event identity connects those actions, and reconciliation ensures convergence. Do not duplicate lifecycle state inside `SecurityEvent` unless an existing pipeline contract requires it.

### Resolve identity before evaluation

Sensor performs Pod-to-deployment resolution using Kubernetes UID and owner relationships. The canonical event retains both the original subject and resolved ACS entity.

### Prefer adapters over a version-specific domain model

Version-specific report adapters produce one internal canonical event. Start with the report API observed in the test fixture, then add `openreports.io/v1alpha1` without changing detection or alert creation.

## Scope

### MVP scope

- Kyverno-produced namespaced reports.
- Results `fail`, `error`, and `warn`.
- Report-level Pod scope.
- Pod UID to ACS Deployment resolution.
- `SECURITY_EVENT` with `PolicyResult` payload matching.
- Security-event source, type, reported policy, reported rule, result, reported severity, and subject-kind criteria.
- Default catch-all Kyverno policy-result ACS policy.
- Standard alert creation, filtering, exclusions, notification compatibility, and resolution.
- Feature-flagged rollout.

### Deferred scope

- Blocked admission attempts for resources that were never created.
- Node, cluster, and arbitrary Kubernetes-resource entities.
- Pass and skip results as alerts.
- ACS context projected back into Kyverno.
- Automatic enforcement or remediation.
- A separate security-events or integration-findings UI.
- Vulnerability enrichment beyond what the standard deployment alert view already provides.
- Kyverno intermediary AdmissionReport and BackgroundScanReport CRDs.

## Canonical domain contract

The exact protobuf shape should follow repository conventions, but the logical contract is:

```text
SecurityEvent
  id
  source
  type
  observed_at

  report
    api_version
    kind
    uid
    name
    namespace
    resource_version

  subject
    api_version
    kind
    uid
    name
    namespace

  resolved_entity
    type
    id

  details (oneof)
    PolicyResult
      reported_policy
      reported_rule
      category
      result
      reported_severity
      message
      properties

    // Deferred until a real adapter is implemented:
    FileIntegrityResult
    SecurityProfileDenial
```

Use typed enums for event type, policy result, reported severity, and entity type. Represent producer source as a normalized, bounded string unless repository compatibility conventions strongly favor an enum. A string permits a standards-based report producer to integrate without requiring a new ACS protobuf release; well-known sources still receive canonical display names. Preserve unknown producer properties inside the typed payload without making arbitrary properties policy criteria by default.

### Source normalization

Adapters must normalize source safely and predictably:

- Preserve the original producer source in details for diagnostics.
- Generate a canonical source used for matching and display, such as `Kyverno`.
- Apply length limits and reject control characters.
- Never infer source solely from free-form message text.
- Prefer an explicit report result source, then well-known management labels, then the adapter identity.
- Record the normalization method in tests, not as customer-visible mutable state.

Adding support for an unknown source must not require changes to the core detector. Adding a new event type or typed payload does require an explicit schema and policy-language decision.

### Alert representation

`SecurityEvent` is an ingestion and evaluation object; `storage.Alert` remains the durable customer object. During Phase 4, decide whether to:

1. Add typed security-event attributes to `storage.Alert.Violation`, or
2. Reuse its existing key/value attributes with a stable, documented key set.

Prefer typed attributes if key/value reuse would make search indexing, API evolution, or UI rendering depend on string conventions. Whichever representation is chosen must preserve source, type, reported policy, reported rule, subject, result, reported severity, and message in the ordinary alert without requiring a parallel user-facing datastore.

### Semantic layers

Keep the terms distinct throughout APIs, metrics, logs, and UI:

| Layer | Term | Kyverno example |
|---|---|---|
| Producer output | Report result | A failed PolicyReport entry |
| Canonical ingestion | Security event | Source Kyverno, type Policy Result |
| Typed evidence | Policy result | Policy, rule, result, message |
| ACS decision | Violation | An ACS policy matched the event |
| Stored/presented object | Alert | Deployment alert in the Violations view |

### Stable identity

Do not use result-array position or observation timestamp. The initial identity input should be:

```text
cluster ID
+ report UID
+ normalized source
+ reported policy name
+ rule name
+ subject UID
```

If fixtures reveal legitimate duplicate results with this tuple, add the smallest stable producer-provided discriminator. Document and test any identity change because it affects alert continuity.

## Phase 0: fixtures, baseline, and seam confirmation

### Implementation

- Capture sanitized fixtures produced by supported Kyverno versions for:
  - a new failing Pod result;
  - multiple failing rules for one Pod;
  - fail-to-pass or fail-to-absent transition;
  - reordered results;
  - report deletion;
  - a report with no actionable results.
- Capture both report API shapes intended for eventual support.
- Trace and document the existing paths for:
  - unstructured CRD discovery and informer registration;
  - Sensor resource reconciliation;
  - Pod-to-deployment ownership lookup;
  - discrete-event policy matching;
  - `AlertResults` delivery and alert resolution.
- Record baseline behavior of the main Violations view using a normal deployment runtime alert.
- Inventory naming collisions with existing Kubernetes Event, audit event, SensorEvent, AlertResults, and ACS policy event-source types.
- Trace how `storage.Alert.Violation` attributes are serialized, indexed, exposed through APIs, and rendered by the main view.

### Validation

- Reproduce fixture generation in a disposable cluster with a pinned Kyverno version.
- Verify where subject identity resides (`scope`, owner reference, or version-specific equivalent).
- Verify whether a corrected workload removes a result, changes it to pass, or replaces the report.
- Measure the delay from workload creation to final report update.

### Checkpoint 0

Proceed when the final report provides a stable Pod UID and observed transitions can be represented as create/update/remove.

Adjust if:

- Final reports do not reliably identify the subject: evaluate owner references or a narrowly scoped Kyverno adapter.
- Report latency is too high for the desired UX: retain the state path but separately investigate admission signals; do not couple the MVP to intermediary CRDs.
- Corrected resources do not converge reliably: make periodic reconciliation mandatory before continuing.
- `SecurityEvent` is too ambiguous in an affected API/package: qualify the package or local Go type rather than changing customer-facing terminology prematurely.

## Phase 1: canonicalization as a pure, versioned adapter

### Implementation

- Define the internal canonical event and typed result/severity mappings.
- Define normalized source and event type independently: `source=Kyverno`, `type=PolicyResult`.
- Implement a pure adapter from the first report API version.
- Read subject information from report-level scope.
- Normalize source and result values without discarding the original strings.
- Return a deterministic collection sorted by canonical ID.
- Diff old and new report objects:
  - new minus old -> create;
  - shared identity with changed content -> update;
  - old minus new -> remove.
- Treat report deletion as removal of every previously actionable result.

### Validation

- Table-driven unit tests using the captured fixtures.
- Property-style tests for:
  - result reordering does not change IDs;
  - repeated processing is idempotent;
  - unknown optional properties do not break parsing;
  - malformed individual results do not discard valid siblings;
  - normalization is deterministic.
- Fuzz the unstructured-object boundary and assert no panics.
- Test source normalization, length bounds, unknown sources, and conflicting source hints.

### Checkpoint 1

Review the canonical contract before adding generated APIs or wiring. Confirm that it contains only producer facts and entity references, not ACS alert-policy decisions. Confirm that adding a future typed payload does not require changing the Kyverno adapter.

Adjust if:

- Identity collisions occur: incorporate a stable discriminator supported by fixtures.
- API versions differ substantially: create explicit adapters behind one interface rather than adding version conditionals throughout the dispatcher.
- Report updates replace results too aggressively: model the entire report as a snapshot and rely on set reconciliation.

## Phase 2: Sensor watch and observable dry-run flow

### Implementation

- Register report CRDs using the established optional-CRD availability pattern.
- Start no informer when the feature flag is disabled.
- Handle CRD absence and removal without degrading Sensor health.
- Feed canonical events into a bounded internal queue or the existing resource-event pipeline.
- Add dry-run diagnostics and metrics before producing alerts:
  - reports observed;
  - actionable results canonicalized;
  - parse failures;
  - create/update/remove operations;
  - processing latency;
  - queue pressure or drops.
- Do not log full producer messages or properties at info level.

### Validation

- Unit tests for availability and informer registration.
- Component test with a fake dynamic client covering add, update, delete, and CRD absence.
- Disposable-cluster test proving that a real Kyverno result reaches the dry-run handler.
- Load a synthetic report set large enough to expose unbounded allocation or excessive logging.

### Checkpoint 2

Demo one real Kyverno failure reaching Sensor with stable identity and correct lifecycle operations. Review report latency and resource cost.

Adjust if:

- Watch volume is excessive: filter persisted result types operationally, reduce retained object data, or evaluate the reports-server path.
- Informer wiring duplicates existing generic CRD infrastructure: refactor only the minimum reusable seam needed by this adapter.
- Metrics reveal frequent malformed results: preserve forward compatibility and expose producer/version diagnostics.

## Phase 3: Pod-to-deployment resolution

### Implementation

- Resolve a report subject by Pod UID first, then namespace/name only as a guarded fallback.
- Traverse the existing Sensor ownership model to the ACS deployment identity.
- Retain the original Pod reference in the canonical event.
- Classify unresolved subjects explicitly instead of silently dropping them.
- Record resolution method and failure reason in bounded metrics.
- Define behavior for terminated Pods whose report remains briefly visible.

### Validation

- Table-driven tests for:
  - Pod -> ReplicaSet -> Deployment;
  - Pod -> StatefulSet;
  - Pod -> DaemonSet;
  - Pod -> Job/CronJob if represented by the ACS deployment model;
  - standalone Pod;
  - missing owner;
  - deleted Pod;
  - namespace/name reuse with a different UID.
- Integration test against Sensor stores with a realistic ownership chain.
- Real-cluster test confirms the resolved ACS deployment ID matches the UI deployment.

### Checkpoint 3

Require a high successful-resolution rate for supported workload types in the fixture and cluster test set. Do not silently convert unresolved Pod findings into deployment alerts.

Adjust if:

- Deleted Pods frequently fail resolution: add a bounded UID-to-deployment tombstone cache with explicit retention.
- Job/CronJob ownership does not map cleanly: defer those types and expose them as unresolved metrics.
- Name fallback produces ambiguity: remove it from production behavior and require UID-based resolution.

## Phase 4: security event source and PolicyResult matching

### Implementation — MVP (Phase 4a)

- Add `SECURITY_EVENT` to the storage policy event-source enum, plus a single minimal field (e.g. Security Event Type) as its only `eventSourceRequirements` entry — see the "Ship a default, built-in policy" guiding decision above. Update the non-exhaustive `EventSource` switch call sites listed there so the new value is actually recognized end to end.
- Define a dedicated detector entry point that accepts the resolved deployment and `SecurityEvent`, matching unconditionally against the one built-in default policy (no field-level criteria beyond the minimal required field).
- Produce standard alert violations containing source, reported policy, reported rule, subject, result, reported severity, and message — packed into the violation message/details even before any of these are independently matchable criteria.
- Ensure security events have no enforcement actions in the initial release.
- Choose and implement the alert-violation representation described in the canonical contract (needed regardless of how much of the criteria language exists).

### Implementation — deferred (Phase 4b, gated on demand)

- Add common field names and metadata beyond the MVP minimum: Security Event Source, Subject Kind.
- Add `PolicyResult` field names and metadata: Reported Policy, Reported Rule, Policy Result, Reported Severity, each independently matchable.
- Restrict common fields to `SECURITY_EVENT` and payload fields to compatible event types during validation.
- Compile a full matcher over `SecurityEvent` with typed `PolicyResult` accessors (today's minimal detector only needs an unconditional match, not a compiled matcher).
- Keep arbitrary `properties` out of the policy language even at this stage; retain them for details and diagnostics only.
- Document policy field display labels independently of protobuf and Go identifiers.

### Validation

- Policy validation tests reject security-event fields on incompatible event sources.
- Policy validation tests reject `PolicyResult` fields for incompatible security-event types.
- Unknown but valid event sources can be matched without recompiling the detector.
- Matcher tests cover exact, multi-value, negated, and multi-group criteria.
- Detector tests prove:
  - matching policies create an alert;
  - nonmatching policies do not;
  - multiple matching reported rules remain distinguishable **as separate violation entries within one alert**, not as separate alerts (see validated finding below);
  - repeated events deduplicate;
  - remove operations resolve the expected alert **once no actionable findings remain for that deployment** (whole-alert resolve, not per-violation — see below).
- Backward-compatibility and generated-proto checks pass.

**Validated finding (`central/detection/alertmanager/alert_manager_impl.go`, `proto/storage/alert.proto`)**: `Alert_Violation` has no independent lifecycle — no per-violation state, and `ResolveAlertRequest` only takes a whole alert ID. Alerts merge by `(PolicyID, ViolationState, Entity)`, so every concurrent Kyverno finding against the same deployment, under the one default policy, accumulates into a **single Alert** (violations list capped at 40, oldest evicted first on merge). Consequence: if a deployment fails two Kyverno rules and only one gets fixed, the alert correctly stays active — resolution is "no active findings remain," matching Definition of done #9's literal wording, but a fixed rule's violation entry can linger in the list until evicted or the whole alert resolves, rather than disappearing immediately. Accepted as correct MVP behavior; revisit only if user testing (Checkpoint 6) shows this is confusing.

### Checkpoint 4

Review policy semantics with UI and policy owners before creating the default policy. Confirm whether `warn` should alert by default and whether reported severity is only match data or also influences ACS severity.

Initial recommendation:

- ACS policy severity controls alert severity.
- Reported severity remains visible and filterable within event criteria/details.
- `fail` and `error` match the default policy.
- `warn` support exists but default behavior is decided using product feedback.

Also decide the alert-detail representation here. Do not proceed with an undocumented string-key contract if the fields need first-class search or API compatibility guarantees.

## Phase 5: first end-to-end standard alert

### Implementation

- Connect the detector output to the existing Sensor `AlertResults` path.
- Use the resolved deployment ID and the existing deployment alert path. Preserve `SecurityEvent.source` and type in violation details without misrepresenting producer provenance as an ACS detector.
- Ensure Central constructs and stores an ordinary `storage.Alert`.
- Preserve original event time while recording normal ACS first/last occurrence semantics.
- Add a feature-flagged default policy named `Reported Kubernetes policy violation` or another product-reviewed name that does not use “external.”
- Give the policy no enforcement actions.

### Validation

- Focused Sensor-to-Central integration test covering create and resolve.
- Verify alert ID stability across identical report resyncs and result reordering.
- Verify the stored alert contains the correct cluster, namespace, Deployment, policy snapshot, lifecycle stage, and violation details.
- Verify no separate security-event datastore is required for the main user-facing path.
- Run existing alert lifecycle and deduplication tests affected by the new source.

### Checkpoint 5: first product demo

Demonstrate:

1. Create a violating Deployment.
2. Observe the Kyverno report.
3. Observe Sensor canonicalization and deployment resolution metrics.
4. Query the standard Central alert API and find the active deployment alert.
5. Correct the Deployment and observe the same alert resolve.

At this checkpoint, decide whether to proceed directly to UI work or first repair lifecycle or identity gaps. Do not expand entity types until create/update/resolve is reliable.

## Phase 6: main Violations view integration

### Implementation

- Render the alert through the existing deployment-violation presentation.
- Show event provenance in violation details:
  - source;
  - event type;
  - reported policy;
  - reported rule;
  - original subject;
  - reported severity and result;
  - message.
- Reuse normal cluster, namespace, deployment, ACS policy, lifecycle, severity, and state filters.
- Add source or reported-policy filters only if the existing search model cannot express the required workflow.
- Add search/index fields required for the agreed workflows:
  - Security Event Source;
  - Security Event Type;
  - Reported Policy;
  - Reported Rule, if justified by expected usage.
- Keep `Policy` as the ACS policy filter; label producer policy explicitly as `Reported Policy` to avoid ambiguity.
- Verify sorting semantics for source and reported policy, including absent values on ACS-native alerts.
- Ensure alert links navigate to the resolved Deployment.

**Validated findings that shape this phase (`ui/apps/platform/src/Containers/Violations/`, `pkg/search/options.go`)**:
- Sorting is entirely server-side: every sortable column's `sortField` must match a `FieldLabel` registered in `pkg/search/options.go`, which drives the actual Postgres `ORDER BY`. Since `Alert.violations` is stored as an opaque protobuf blob (`search:"-"`, no Postgres column of its own), **Reported Policy/Rule/Severity/Source must be promoted to real top-level `search:`-tagged fields on `storage.Alert`** (not left inside the violation message) before they can be sorted or used as a workflow-view filter. This is a small, targeted proto + schema + search-option change — separate from, and does not require, the Phase 4b customer-criteria language.
- The "Platform"/"Node" split some teams call "tabs" is actually a shared `NavList` sub-nav (`filteredWorkflowView`, also reused by the Risk page), where each item maps to a hardcoded `SearchFilter` on existing fields (`Entity Type`, `Platform Component`) — not a literal `Tabs` component (that's reserved for the separate Active/Resolved/Attempted `ViolationState` tabs). A future "Security Events" nav item is a small, additive UI change (one literal in `FilteredWorkflowViewSelector/types.ts`, one route, one `NavigationItem`, one `switch` case) once it has a real backend field to filter on — decoupled from Phase 4b, can land as soon as the search-field promotion above is done.

### Validation

- UI unit/component tests for rendering and filtering.
- API and PostgreSQL search tests for new alert search fields.
- Mixed-result tests containing ACS-native and security-event-derived alerts.
- Browser-level validation against a real or fixture-backed Central.
- Accessibility and truncation tests for long producer messages and reported policy names.
- Confirm the alert is visible in existing totals and does not require a separate page.

### Checkpoint 6: user workflow review

Have a user complete these tasks without implementation knowledge:

- Find all active Kyverno-derived violations.
- Filter them to one cluster and namespace.
- Open the affected Deployment.
- Identify the original Kyverno policy and rule.
- Determine whether the violation is still active.
- Distinguish the ACS policy from the Kyverno reported policy without documentation.

Adjust labels, details, and filters based on observed confusion rather than adding a new view preemptively.

## Phase 7: exclusions, scopes, notifications, and reconciliation

### Implementation

- Apply existing deployment policy scope and exclusion logic before alert emission.
- Confirm namespace, deployment, image, and expiration-based exclusions that are meaningful for runtime deployment alerts.
- Exercise existing notifier selection without adding producer-specific notification code.
- Add Sensor reconnect reconciliation for active canonical IDs.
- Reject or ignore stale report resource versions where the pipeline can observe them.
- Bound any tombstone or reconciliation state.

### Validation

- End-to-end tests for:
  - namespace exclusion;
  - deployment exclusion;
  - exclusion expiration;
  - scoped policy inclusion;
  - notifier invocation;
  - Sensor disconnect during report update;
  - reconnect with a result removed while disconnected;
  - report deletion;
  - duplicate delivery;
  - out-of-order stale update.
- Restart Sensor and Central independently and verify convergence.

### Checkpoint 7: reliability gate

Proceed to broader rollout only when repeated restart and reconnect testing produces no duplicate active alerts and no stale alerts beyond the documented convergence interval.

If convergence cannot use the standard alert pipeline safely, introduce the minimum durable security-event index required for reconciliation; do not expose it as a second user-facing model.

## Phase 8: compatibility, scale, and rollout

### Implementation

- Add the second report API adapter, expected to be `openreports.io/v1alpha1`.
- Detect available APIs independently and prevent duplicate ingestion when a producer writes both formats.
- Add component capability negotiation if required for Central/Sensor version skew.
- Define feature-flag upgrade and downgrade behavior.
- Add support bundle diagnostics for:
  - discovered report APIs;
  - watched report count;
  - canonicalization failures;
  - resolution rate;
  - active event count;
  - reconciliation status.

### Validation

- Compatibility matrix:
  - supported Kyverno versions;
  - each supported report API;
  - reports absent;
  - both report APIs present;
  - Sensor older than Central;
  - Central older than Sensor, where supported.
- Scale tests with representative numbers of clusters, reports, results, and update bursts.
- Measure Sensor CPU/memory, API watch traffic, Central ingest rate, alert storage growth, and UI query latency.
- Security review untrusted strings, property sizes, RBAC, scoped access, and log handling.

### Checkpoint 8: rollout decision

Roll out in stages:

1. Development-only feature flag.
2. Internal or selected-cluster dogfood.
3. Opt-in customer preview with diagnostics.
4. Default-on ingestion with default policy behavior decided from preview data.
5. Remove the feature flag only after upgrade, scale, and support evidence is complete.

Pause or narrow rollout if:

- report watch load materially affects cluster API performance;
- deployment resolution is below the agreed threshold;
- alert churn or duplication is visible;
- default policy volume overwhelms the main Violations view;
- Kyverno/report-version compatibility requires producer-specific semantics in the core detector.

## Expansion checkpoints and additional adapters

Expansion should be evidence-driven and performed one entity, source, or typed payload at a time. Adding an adapter must not require producer-specific conditionals in the core detector.

### File Integrity Operator

Use `FileIntegrityNodeStatus` as the authoritative current-state input and canonicalize relevant results as:

```text
SecurityEvent
  source: OPENSHIFT_FILE_INTEGRITY_OPERATOR
  type: FILE_INTEGRITY_RESULT
  details: FileIntegrityResult
  subject: Node
```

Map Kubernetes node identity to an ACS Node and use the standard Node alert path. Treat a clean subsequent scan as resolution. Treat scan execution errors separately from actual file-integrity changes. Kubernetes Events may supplement transition detail but must not be authoritative reconciliation state.

Gate implementation on the Kyverno adapter proving that a second typed payload can reuse transport, entity resolution, policy evaluation, and alert creation without weakening type safety.

### Security Profiles Operator

Separate three SPO concerns:

- Runtime seccomp/AppArmor/SELinux denials become `SecurityProfileDenial` events and map to Deployment when Pod/container identity is available, otherwise Node.
- Profile installation/readiness failures are resource or platform health results and require a product decision before entering the main Violations view.
- Profile inventory and recording outputs are posture context, not violations by default.

Before implementation, capture supported structured denial output and decide whether Collector, Sensor, or another supported stream owns ingestion. Do not infer runtime denials solely from profile CRD readiness.

### Node-scoped policy results

Add when real reports contain reliable Node UID or Kubernetes node-name scope. Reuse the standard Node alert path and validate node-specific scopes and UI behavior.

### Cluster and generic resource events

First decide whether ACS needs a first-class Cluster alert entity. Do not model cluster findings as fake deployments. Generic Resource may be an interim representation only if filtering and UI semantics remain truthful.

### Blocked admission attempts

Treat these as event history rather than report state. Investigate the existing Kubernetes audit path or supported producer output. Kubernetes `Event` objects are best-effort and should be supplemental. Do not consume Kyverno intermediary report CRDs by default.

### Vulnerability and runtime enrichment

After standard deployment alerts work, verify which enrichment is already available through the normal alert and deployment views. Add only missing context that materially improves prioritization.

## Validation environments

Maintain three layers:

### Fast local tests

- Pure adapter, identity, diff, matcher, resolution, and policy-validation tests.
- Must run without Kubernetes or PostgreSQL.

### Component integration tests

- Fake dynamic client and Sensor stores.
- Sensor-to-Central alert lifecycle.
- PostgreSQL-backed alert queries where storage behavior changes.

### Disposable-cluster scenario

- Pinned Kubernetes and Kyverno versions.
- Installs ACS components required for the flow.
- Creates, updates, and deletes a known violating Deployment.
- Captures report, metrics, Central API result, and final resolution as artifacts.
- Runs on demand initially; promote to periodic CI after stability is demonstrated.

## Operational success measures

- Percentage of actionable Pod-scoped results resolved to an ACS deployment.
- P50/P95 time from report resource version to visible Central alert.
- Duplicate alert rate after report updates and Sensor reconnects.
- Stale active alert count after convergence interval.
- Sensor CPU and memory per 1,000 watched reports.
- Canonicalization error rate by API version and producer.
- Number of alerts suppressed by scopes and exclusions.
- Main Violations query latency and alert-volume change.

Initial thresholds should be recorded during dogfood rather than invented before measurement. Promotion between rollout stages requires explicit agreed thresholds.

## Pull request decomposition

Keep changes independently reviewable and reversible:

1. Fixtures and pure canonicalizer with identity/diff tests.
2. Optional CRD discovery and dry-run metrics.
3. Pod-to-deployment resolver and tests.
4. Policy enum, criteria metadata, validation, and matcher.
5. Security-event detector with `PolicyResult` matching producing standard `AlertResults`.
6. Central standard-alert lifecycle and reconciliation integration.
7. Default policy behind the feature flag.
8. Main Violations rendering and workflow tests.
9. Exclusions, notifier, restart, and reconnect validation.
10. OpenReports adapter, compatibility diagnostics, and scale hardening.

Avoid combining generated protobuf output, Sensor wiring, Central lifecycle changes, and UI changes into one initial pull request.

## Immediate next actions

Done (PR #1 — `pkg/policyreport/`, fixtures + pure canonicalizer):

1. ~~Validate the current prototype against a real Kyverno report and correct it to use report-level scope.~~ Done against real Kyverno v1.18.2 output — corrected the schema assumption to a single top-level `scope` field per report object (see `pkg/policyreport/testdata/raw/README.md`), not the `results[].resources[]` array originally assumed here.
2. ~~Replace result-index identity with semantic stable identity.~~ Done (`pkg/policyreport/identity.go`).
3. ~~Add old/new snapshot diff tests before adding more wiring.~~ Done (`pkg/policyreport/diff.go` + tests).
4. ~~Locate and document the exact existing Pod-to-deployment resolver to reuse.~~ Done — `references.ParentHierarchy.TopLevelParents(podUID)` + `DeploymentStore`, both exposed via `StoreProvider`.
5. ~~Trace the smallest existing discrete-event detector path that produces a deployment `AlertResults` message.~~ Done — the file-access path in `sensor/common/detector/detector.go` is the template (already conditionally sets `AlertResults.Source`).
7. ~~Produce the Phase 0 disposable-cluster fixtures and record report convergence behavior.~~ Done — 3 real fixtures captured from a live cluster (see `pkg/policyreport/testdata/raw/`).

Superseded: item 6 below (draft `SecurityEvent`/`PolicyResult` protobuf) is deferred, not skipped — see "Ship a default, built-in policy" above. The MVP no longer needs the full field vocabulary drafted up front, only the minimal shape needed for Phase 4a.

Next (PR #2 onward):

1. Register the `wgpolicyk8s.io/v1alpha2` CRDs in Sensor behind a feature flag (Phase 2), copying the VirtualMachine/KubeVirt availability-checker + CRD-watcher pattern.
2. Wire Pod-to-Deployment resolution using the already-identified existing mechanism (Phase 3) — no new resolver needed.
3. Add `SECURITY_EVENT` to `storage.EventSource` with the Phase 4a minimal shape (one built-in policy, no customer criteria yet), updating the non-exhaustive `EventSource` switch call sites enumerated above.
4. Build the Phase 5 detector path (template: file-access) and confirm the whole-alert-resolve behavior documented in Phase 4's validation section is acceptable end to end.
5. Promote Reported Policy/Rule/Severity/Source to real `search:`-tagged `Alert` fields (Phase 6) so they're sortable/filterable — independent of, and can land before, any Phase 4b customer-criteria work.

## Terminology decision record

The plan intentionally uses:

- `SecurityEvent` for the source-neutral canonical envelope.
- `PolicyResult` for Kyverno/OpenReports semantics.
- `Security Event` for the ACS policy event source.
- `Source` for producer provenance.
- `Reported Policy`, `Reported Rule`, and `Reported Severity` for producer-supplied values.
- `Violation` and `Alert` only after ACS policy evaluation.

It intentionally avoids `external`, `third-party`, `finding`, and `cluster security event`. Revisit this terminology only if implementation reveals a concrete collision or user research shows misunderstanding.

## Related files

- `issues/issue_external_policy_report_ingestion.md`
- `projects/policy-engine-evolution/spec.md`
- `pkg/policyreport/`
- `sensor/kubernetes/listener/resources/policyreport/`
- `sensor/kubernetes/listener/watcher/policyreport/`
- `central/externalpolicyviolation/` (prototype to be renamed or replaced)
- `proto/internalapi/central/external_policy_violation.proto` (prototype to be replaced by the agreed domain contract)
- `proto/storage/external_policy_violation.proto` (prototype; a separate user-facing store is not the MVP)
- `proto/storage/policy.proto`
- `proto/storage/alert.proto`
- `pkg/booleanpolicy/`
