# Risk classification proof-of-concept plan

## Goal

Implement an incremental proof of concept for semantic workload risk classification, while preserving the existing risk score and priority behavior until classification is independently verified.

The MVP adds a class/case-like semantic layer to deployment risk processing with minimal API disruption. The initial user-visible goal is better sorting and clearer prioritization, not a complete RiskCase data model.

## Working assumptions

- This is a POC and can make additive proto/API/storage changes if needed.
- Existing `Risk.score` semantics must remain unchanged until classification has unit-test coverage and is exposed for validation.
- Classification should be computed during existing deployment risk processing, preferably without adding broad per-deployment raw-data queries.
- If richer image vulnerability signals are needed, add compact image-risk summary data where image risk is computed and reuse it during deployment classification.
- Initial classifier should be deterministic and explainable. It should generate one primary class plus supporting reasons/signals where practical.
- The first iteration may be approximate; the goal is to prevent posture/hygiene volume from outranking obvious urgent risk classes.

## Open questions / decisions to confirm later

1. Naming: use `RiskPriorityClassification`, `RiskPriorityClass`, or `RiskDisposition`? Initial plan uses `RiskPriorityClassification` unless a better existing convention appears.
2. Enum values: action-oriented labels (`RESPOND_NOW`, `INVESTIGATE_NOW`, `REMEDIATE_NOW`, `REMEDIATE`, `IMPROVE_POSTURE`) versus descriptive labels (`ACTIVE_THREAT`, `EXPOSED_EXPLOITABLE`, etc.). Initial plan uses action-oriented values because they map directly to UX.
3. Whether to expose reasons in API/UI during the POC. Initial plan stores/exposes classification first; reasons are optional if implementation remains small.
4. Whether the final in-place score modification should encode priority bands into `Risk.score` or adjust multiplier logic. Initial plan prefers feature-flag-free POC scoring changes only after classification tests pass, but will choose the least invasive path supported by existing rank/sort code.

## Classification model MVP

Primary ordering, from highest priority to lowest:

1. `RESPOND_NOW`
   - Confirmed/critical active runtime threat where available.
   - MVP approximation may use critical active runtime policy alerts once richer alert metadata is available.
2. `INVESTIGATE_NOW`
   - Suspicious runtime activity, especially process baseline violations.
3. `REMEDIATE_NOW`
   - External exposure combined with critical/exploitable vulnerability signal.
   - External exposure combined with powerful identity or sensitive credentials.
   - Public privileged workload.
4. `REMEDIATE`
   - Cluster-wide identity / secret-read blast radius.
   - Internal critical/high material vulnerabilities.
   - Dangerous privilege without known external path.
5. `IMPROVE_POSTURE`
   - CVE volume, image age, component count, risky utilities, low-severity policy hygiene, internal port count, and other backlog/posture-only issues.
6. `UNKNOWN`
   - Insufficient data or no classification.

## Incremental implementation checklist

### Phase 0: Baseline and characterization

- [ ] Review current risk proto/storage/API flow for `storage.Risk` and deployment list/details responses.
- [ ] Review existing generated code expectations and codegen requirements for proto changes.
- [ ] Review current `risk_demo` tests and keep them as characterization tests for existing formula failures.
- [ ] Add/extend curated workload unit tests that assert current bad ordering as a baseline, with readable logged multipliers/classes.

### Phase 1: Curated workload tests for target behavior

- [ ] Create a test fixture representing curated workloads similar to the YAML demo:
  - benign noisy posture backlog
  - external Spring4Shell/critical-RCE-like workload with credential secret
  - critical runtime alert / baseline deviation workload
  - cluster-wide secret reader
  - public privileged critical RCE
  - internal CVE/noise workload
  - attack-path pair if feasible at unit level, or document why it is not representable yet
- [ ] Add assertions for desired classification order independent of existing raw score.
- [ ] Add tests that verify raw score remains unchanged before scoring-sort changes.

### Phase 2: Add storage/API surface for classification

- [ ] Add a classification enum to the appropriate proto, likely `proto/storage/risk.proto` unless a deployment-specific type is more appropriate.
- [ ] Add a new field on `storage.Risk`, likely `priority_classification`.
- [ ] Regenerate protobuf code as needed.
- [ ] Update risk construction and persistence paths to tolerate unset/unknown classification.
- [ ] Add unit tests verifying classification is persisted in deployment risk results.

### Phase 3: Image risk signal summaries if required

- [ ] Determine whether current image risk results expose enough vulnerability detail for classification.
- [ ] If not, add compact vulnerability summary fields to image risk data, for example:
  - max severity
  - severity counts
  - has critical vulnerability
  - has important vulnerability
  - has network-attack-vector vulnerability if available
  - has KEV / known exploit / high EPSS if available in image/CVE data
- [ ] Compute these summaries during image risk scoring, not by fetching full image scan data during deployment scoring.
- [ ] Add unit tests that build synthetic images/CVEs and verify the summary.

### Phase 4: Implement deployment risk classifier

- [ ] Add a small classifier package or file under `central/risk/scorer/deployment` or `central/risk/classification`.
- [ ] Classifier inputs should initially be bounded and already available:
  - deployment port exposure
  - service account permission level
  - automounted service account token
  - mounted secrets
  - privileged/security context flags
  - policy risk results / alert summary
  - process baseline risk results
  - image risk summaries
- [ ] Implement deterministic classification rules in priority order.
- [ ] Store classification on the new `Risk` field during deployment risk scoring/processing.
- [ ] Add tests for each classification rule and edge cases.
- [ ] Add curated workload tests verifying target classes.

### Phase 5: Verify classification while preserving current score/priority

- [ ] Ensure existing `Risk.score` calculations are unchanged.
- [ ] Ensure existing ranker/priority behavior is unchanged.
- [ ] Ensure new classification can be read from the risk object/API response.
- [ ] Run relevant unit tests:
  - deployment risk scorer tests
  - image risk scorer tests, if changed
  - risk manager/datastore tests, if changed
  - curated POC tests

### Phase 6: Improve sorting by minimally patching existing score behavior

The POC should not rely on a separate hidden sort score or banded score unless absolutely necessary. The intended PM-facing path is to keep the classification system as the semantic layer, then minimally tweak the existing formula so intra-class and legacy raw-score ordering become less arbitrary and less misleading.

Candidate score patches to evaluate and implement incrementally:

- [ ] Rework image vulnerability contribution so low/moderate CVE volume cannot dominate indefinitely.
  - Any number of Low severity CVEs should have a hard maximum contribution cap.
  - Moderate severity should likely have a cap as well, higher than Low but lower than Important/Critical.
  - Critical/Important contribution values should be shifted to be more reasonable relative to Low/Moderate volume.
  - If EPSS/KEV/known-exploit metadata is available in image risk processing, incorporate it as a dominant contribution modifier.
- [ ] Rework service reachability contribution so exposure semantics beat port count.
  - External/Route exposure should count for substantially more than an internally exposed port.
  - Consider using max exposure plus small capped additional-port contribution rather than pure sum over ports.
  - Ensure many internal declared ports cannot outscore one meaningful external exposure.
- [ ] Evaluate whether low-severity policy violations should have a capped posture contribution.
- [ ] Evaluate whether service-account blast radius, automounted token, and privileged/external combinations need a minimal direct multiplier or are sufficiently handled by classification for the POC.
- [ ] Implement score patches one at a time behind small helper functions and comments.
- [ ] Add tests proving each score patch moves ordering in the intended direction without relying solely on classification:
  - low/moderate CVE volume is capped and cannot bury a single critical/exploitable vulnerability as easily.
  - external exposure outranks internal port count in the reachability multiplier.
  - `REMEDIATE_NOW` workloads get more reasonable raw scores than posture-only backlog workloads where possible.
  - cluster-wide identity is not buried behind posture backlog if a minimal direct score patch is added.

### Phase 7: UI updates

- [ ] Update frontend types/services to include classification.
- [ ] Replace or de-emphasize `Priority` with `Risk class` where appropriate.
- [ ] Add class labels/icons in the Risk table.
- [ ] Keep raw score visible only if useful for intra-class sorting/debugging.
- [ ] Change default UI sort to classification-based ordering if API/search sorting supports it; otherwise rely on improved backend score sorting.
- [ ] Add UI unit tests / component tests for rendering labels and sorting behavior where feasible.

### Phase 8: Final verification and handoff

- [ ] Run targeted Go unit tests.
- [ ] Run targeted UI lint/tests if UI changed.
- [ ] Update this plan with completed items and notable deviations.
- [ ] Summarize implementation, known limitations, and cluster-deployment validation steps.

## Known limitations for MVP

- Full attack-path cases likely require graph/path data not currently available in deployment risk scorer inputs.
- True public internet reachability is not guaranteed by Kubernetes exposure alone; classify as external exposure rather than proven public reachability.
- Secret materiality may initially be generic unless Secret metadata is plumbed in.
- Alert semantics may initially be limited by current alert search projections.
- Classification is a bridge toward RiskCase/RemediationOpportunity modeling, not a replacement for that fuller design.

## Progress log

- [x] Plan created.
- [x] Added `RiskPriorityClassification` and compact `RiskVulnerabilitySummary` to `storage.Risk`.
- [x] Generated Go and vtproto bindings for `storage/risk.proto` and updated storage protolock.
- [x] Added image vulnerability summary computation during image risk scoring.
- [x] Added deployment risk classifier and store classification on deployment `Risk` objects.
- [x] Added curated workload classification unit tests.
- [x] Implemented initial score patches for image vulnerability volume caps, exposure-dominant reachability, and direct service-account identity blast-radius scoring.
- [x] Added preliminary Risk detail UI rendering for the new risk classification returned by `GetDeploymentWithRisk`.
- [x] Plumbed risk classification into `Deployment`/`ListDeployment`, deployment ranker metadata, deployment list responses, and Risk table rendering/sorting.
- [x] Verified all `./central/risk/...` Go tests pass with isolated module/cache settings.
- [x] Verified `risk_demo` build-tag deployment scorer demo tests pass and print updated computed multipliers.
- [x] Verified targeted UI prettier/eslint for edited Risk detail files.
