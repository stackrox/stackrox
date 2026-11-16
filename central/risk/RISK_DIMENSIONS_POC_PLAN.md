# Risk dimensions proof-of-concept plan

## Goal

Retain the existing single top-level Risk class / disposition for tenant-agnostic triage ordering:

1. Respond now
2. Investigate now
3. Remediate now
4. Remediate
5. Improve posture

Add three explanatory dimensions that can be returned by APIs, persisted on deployments, sorted in list views, and shown/hidden in the Risk table:

1. Activity — observed runtime/security behavior.
2. Exposure — unrealized material exposure / blast radius.
3. Posture — hygiene/security-debt backlog.

The top-level disposition remains the default queue for “what needs attention now.” The three dimensions explain why a workload landed there without forcing users to mentally combine multiple independent columns.

## Non-goals

- Do not replace the top-level Risk class with three independent top-level fields.
- Do not remove existing columns currently used in the demo: Priority, Old score, New score, Risk class.
- Do not remove existing ability to sort by current/demo columns.
- Do not introduce full RiskCase entities in this iteration.
- Do not add graph/path/Secret materiality dependencies unless already available from current risk processing.

## Proposed API/model shape

### Existing retained top-level field

Keep the existing primary field:

```proto
RiskPriorityClassification risk_priority_classification
```

This remains the primary disposition / attention queue.

### New dimension fields

Add new enum fields to `storage.Risk`, `storage.Deployment`, and `storage.ListDeployment`.

Suggested enums:

```proto
enum RiskActivityClassification {
  RISK_ACTIVITY_UNKNOWN = 0;
  RISK_ACTIVITY_RESPOND_NOW = 1;
  RISK_ACTIVITY_INVESTIGATE_NOW = 2;
  RISK_ACTIVITY_NONE_OBSERVED = 3;
}

enum RiskExposureClassification {
  RISK_EXPOSURE_UNKNOWN = 0;
  RISK_EXPOSURE_REMEDIATE_NOW = 1;
  RISK_EXPOSURE_REMEDIATE = 2;
  RISK_EXPOSURE_NONE_DETECTED = 3;
}

enum RiskPostureClassification {
  RISK_POSTURE_UNKNOWN = 0;
  RISK_POSTURE_IMPROVE = 1;
  RISK_POSTURE_NONE_DETECTED = 2;
}
```

Sort order should not rely on enum numeric order because `UNKNOWN = 0` must sort last for default ascending sort. Add persisted sort-order integer fields on `Deployment` / `ListDeployment` for list sorting:

```proto
int32 risk_activity_sort_order = ...;
int32 risk_exposure_sort_order = ...;
int32 risk_posture_sort_order = ...;
```

Suggested sort orders:

Activity:

```text
1  Respond now
2  Investigate now
3  No observed suspicious activity
99 Unknown
```

Exposure:

```text
1  Remediate now
2  Remediate
3  No material exposure detected
99 Unknown
```

Posture:

```text
1  Improve posture
2  No significant posture finding
99 Unknown
```

## Classification semantics

The classifier should compute dimensions first, then derive the top-level disposition from them.

### Activity

Activity is driven by observed runtime/security behavior.

- Respond now:
  - confirmed threat policy names/signals, for example crypto mining / malware.
- Investigate now:
  - suspicious runtime policy names/signals, for example nmap/netcat/shell/runtime anomalies.
  - process baseline deviations.
- None observed:
  - no runtime/process/policy signal that belongs to Respond or Investigate.
- Unknown:
  - reserved for insufficient data / unset values.

### Exposure

Exposure is driven by unrealized material exposure and blast radius.

- Remediate now:
  - external exposure + critical / KEV / ransomware / high-EPSS vulnerability.
  - external exposure + powerful identity.
  - external exposure + dangerous privilege.
  - external exposure + mounted Secret + important/critical vulnerability.
- Remediate:
  - cluster-wide or cluster-admin identity without current external path.
  - dangerous privilege without current external path.
  - internal critical / KEV / ransomware vulnerability.
- None detected:
  - no material exposure / blast-radius condition currently detected.
- Unknown:
  - unset/insufficient data.

### Posture

Posture is driven by hygiene/backlog/security-debt signals.

- Improve posture:
  - vulnerability volume without material exposure.
  - old image.
  - component count/risky utilities.
  - low/medium hygiene policy violations.
  - root/no resource limits/latest tag/package manager/no capabilities dropped/etc.
- None detected:
  - no posture signals in current risk results.
- Unknown:
  - unset/insufficient data.

### Top-level disposition derivation

Derive the existing `RiskPriorityClassification` from dimensions using this precedence:

```text
Activity Respond now       -> Respond now
Activity Investigate now   -> Investigate now
Exposure Remediate now     -> Remediate now
Exposure Remediate         -> Remediate
Posture Improve posture    -> Improve posture
Otherwise                  -> Unknown or lowest available no-finding state
```

This preserves a single default attention ordering while allowing the UI/detail page to explain whether the classification came from observed activity, material exposure, posture debt, or a combination.

## Implementation checklist

### Phase 1: Tests first

- [ ] Add classifier unit tests for dimension outputs and derived top-level disposition.
- [ ] Cover current demo workloads:
  - `respond-crypto-miner`: Activity Respond now, Exposure none, Posture optional/none, top Respond now.
  - `investigate-network-scanner`: Activity Investigate now, Exposure none, Posture optional/none, top Investigate now.
  - `danger-spring4shell-internet`: Activity none, Exposure Remediate now, Posture optional, top Remediate now.
  - `danger-cluster-secret-reader`: Activity none, Exposure Remediate, Posture optional/none, top Remediate.
  - `policy-smell-pile`: Activity none, Exposure none, Posture Improve, top Improve posture.
  - `cve-noise-batch`: Activity none, Exposure none, Posture Improve, top Improve posture.
- [ ] Add tests for top-level precedence when more than one dimension is non-empty:
  - Activity Investigate now + Exposure Remediate now => top Investigate now.
  - Activity none + Exposure Remediate + Posture Improve => top Remediate.
- [ ] Add sortfield tests for the three new columns and confirm unknown-last behavior.

### Phase 2: Proto/model additions

- [ ] Add three dimension enums to `proto/storage/risk.proto` or another shared storage proto location.
- [ ] Add fields to `storage.Risk`:
  - `activity_classification`
  - `exposure_classification`
  - `posture_classification`
- [ ] Add fields to `storage.Deployment` and `storage.ListDeployment`:
  - dimension classification values
  - dimension sort-order integers
- [ ] Regenerate protobuf, vtproto, Swagger/GraphQL/generated files as required by existing branch conventions.
- [ ] Update frontend generated TypeScript proto types.

### Phase 3: Sort-order helpers

- [ ] Extend existing `pkg/risk` helper(s) with:
  - `ActivityClassificationSortOrder`
  - `ExposureClassificationSortOrder`
  - `PostureClassificationSortOrder`
- [ ] Add tests for all sort-order helpers.
- [ ] Ensure unknown/unset returns `99`.

### Phase 4: Classifier refactor

- [ ] Refactor `central/risk/scorer/deployment/classifier.go` so it returns a struct like:

```go
type classificationResult struct {
    priority storage.RiskPriorityClassification
    activity storage.RiskActivityClassification
    exposure storage.RiskExposureClassification
    posture  storage.RiskPostureClassification
}
```

- [ ] Keep existing top-level behavior stable where possible.
- [ ] Move current Respond/Investigate logic into Activity computation.
- [ ] Move RemediateNow/Remediate logic into Exposure computation.
- [ ] Move vulnerability volume/image age/component count/risky utility/hygiene policy logic into Posture computation.
- [ ] Derive the existing priority classification from the three dimensions.
- [ ] Add/adjust unit tests.

### Phase 5: Persistence and list API

- [ ] During risk processing, persist dimension classifications on `storage.Risk`.
- [ ] Copy dimension classifications from risk to `storage.Deployment` during deployment risk processing.
- [ ] Compute and persist dimension sort orders on `storage.Deployment` before upsert.
- [ ] Regenerate/update deployment Postgres schema/store bindings for the new scalar fields.
- [ ] Update list-deployment DB view to select:
  - activity classification
  - exposure classification
  - posture classification
  - activity sort order
  - exposure sort order
  - posture sort order
- [ ] Scan nullable existing rows safely after in-place upgrade:
  - null enum => unknown
  - null sort order => 99
- [ ] Add/update datastore tests for projection and null-safe scan behavior.

### Phase 6: Search/sort integration

- [ ] Add search option labels:
  - `Risk Activity Classification`
  - `Risk Activity Sort Order`
  - `Risk Exposure Classification`
  - `Risk Exposure Sort Order`
  - `Risk Posture Classification`
  - `Risk Posture Sort Order`
- [ ] Map user-facing dimension columns to sort-order fields in `pkg/search/sortfields/fields_map.go`.
- [ ] Keep existing fields sortable:
  - Priority
  - Old score / Deployment Risk Score
  - New score / Proposed Deployment Risk Score
  - Risk class / Risk Priority Classification
- [ ] Add sort tests verifying:
  - Activity sort uses activity sort order.
  - Exposure sort uses exposure sort order.
  - Posture sort uses posture sort order.
  - Risk class still sorts by top-level sort order and proposed score as currently intended.

### Phase 7: UI table columns and column management

- [ ] Extend `ListDeployment` TypeScript type with the three dimension fields.
- [ ] Add table columns:
  - Activity
  - Exposure
  - Posture
- [ ] Make all three columns managed by existing column management.
- [ ] Preserve current columns and current visibility choices:
  - Name
  - Created
  - Cluster
  - Namespace
  - Priority
  - Old score
  - New score
  - Risk class
- [ ] Suggested default visibility:
  - Keep `Risk class` visible.
  - Keep `Priority` and `Old score` visible for demo comparison.
  - Keep `New score` hidden by default if that is current behavior.
  - Add Activity/Exposure/Posture hidden by default initially, or visible if the demo specifically wants explanatory columns in the table.
- [ ] Ensure each new column can be shown/hidden and sorted.
- [ ] Add compact labels for each dimension.

### Phase 8: Detail-page explanation

- [ ] Update risk indication/details view to show a categorical breakdown:
  - Activity
  - Exposure
  - Posture
- [ ] Keep detailed risk factors below the categorical summary.
- [ ] Do not overload the expanded/table row with full risk factors unless a separate expandable-row task is requested.

### Phase 9: Verification

- [ ] Run backend tests:

```bash
go test ./central/risk/... \
        ./central/deployment/datastore/internal/store/postgres \
        ./central/deployment/datastore \
        ./pkg/search/sortfields \
        ./central/ranking \
        ./central/graphql/resolvers
```

- [ ] Run demo tests:

```bash
go test -tags risk_demo ./central/risk/scorer/deployment -run TestCurrentRiskModelDemo -v
```

- [ ] Run frontend formatting/lint for touched files:

```bash
cd ui/apps/platform
npx prettier --write <touched-ui-files>
npx eslint <touched-ui-files>
```

## Assumptions unless changed

1. Keep the existing top-level `RiskPriorityClassification` as the primary table sort / disposition.
2. Add three explanatory dimensions in addition to the top-level class, not instead of it.
3. Use simple MVP dimension values rather than high/medium/low levels for posture/exposure.
4. Unknown sorts last via explicit persisted sort-order fields.
5. New dimension columns are sortable and column-managed.
6. Do not add per-row Risk datastore lookups to deployment list rendering.
7. Do not remove current demo columns or sorts.

## Clarifications resolved

1. Activity/Exposure/Posture should be hidden by default, but available through column management.
2. Posture should use levels instead of a binary state. Use Critical/High/Medium/Low/None/Unknown if the classifier can justify the thresholds from existing score factors.
3. Keep the top-level column label as `Risk class` for now.

## Progress

- [x] Plan created with resolved assumptions.
- [x] Added Activity, Exposure, and leveled Posture model fields to risk/deployment/list deployment protos.
- [x] Added sort-order helpers and tests for Risk class plus Activity/Exposure/Posture unknown-last ordering.
- [x] Refactored deployment classifier to compute dimensions first and derive the top-level Risk class.
- [x] Persisted dimensions and sort orders through risk processing and deployment storage/list projection.
- [x] Added sortable search fields for Activity, Exposure, and Posture.
- [x] Added UI table columns hidden by default via column management.
- [x] Added detail-page categorical breakdown under Risk indicators.
- [x] Ran backend and UI validation commands listed in Phase 9, except full project codegen remains subject to local scanner cache constraints.
