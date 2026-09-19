# Raw PolicyReport fixtures

Captured live from a real cluster, not hand-authored. Provenance:

- Cluster: `infractl`-provisioned OpenShift 4 (`rc-devloop-test`), Kubernetes v1.35.5, CRI-O 1.35.4 (RHAOS 4.22).
- Kyverno: Helm chart `kyverno/kyverno` 3.8.2, app version v1.18.2, installed with default values (admission, background, cleanup, reports controllers).
- PolicyReport API: `wgpolicyk8s.io/v1alpha2` (the only version this Kyverno version serves; confirmed via `oc get crd policyreports.wgpolicyk8s.io -o jsonpath='{.spec.versions[*].name}'`).
- Test `ClusterPolicy` (`require-secure-pod-security-context`, `validationFailureAction: Audit`) with two rules: `disallow-latest-tag` (image tag must not be `latest`) and `require-team-label` (pod must carry a `team` label), annotated `policies.kyverno.io/severity: high`.

## Real behavior these fixtures capture (not assumed from docs)

- **One `PolicyReport` object per resource**, linked via `ownerReferences`, identified by a top-level `scope` field (`apiVersion`/`kind`/`name`/`namespace`/`uid`) — **not** a `results[].resources[]` array as the original plan draft speculated. Kyverno creates separate report objects for the Pod, its ReplicaSet, and its Deployment independently (via "autogen" rules for the higher-level controllers).
- `results[]` entries carry `policy`, `rule`, `result` (`pass`/`fail`/etc.), `message`, `scored`, `source: kyverno`, a `timestamp`, and — only when set via the `policies.kyverno.io/severity` annotation on the source `ClusterPolicy` — `severity`. It's absent if the policy doesn't set it.
- A rolling Deployment update that fixes a violation does **not** update the Pod-scoped report in place — the old Pod (and its report, via owner-reference garbage collection) is deleted and an entirely new Pod (new UID) gets a brand new report. In-place updates only happen for reports scoped to a resource whose UID doesn't change (e.g. the Deployment itself).
- Deleting a Deployment removes its own and its ReplicaSet's reports promptly; the terminating Pod's report lags slightly behind actual pod termination.
- A ReplicaSet that's scaled to 0 (but not deleted, e.g. kept for rollback history after a rolling update) **keeps its stale report** — report lifetime tracks resource existence, not liveness/replica count. See `03-after-deployment-deletion.yaml`, entry `e31df1d7-...` (`fail-tag-only-7f68f8847`, the pre-rollout ReplicaSet, `DESIRED: 0`).

## Files

1. `01-new-failures-and-multi-rule.yaml` — two Deployments just created: one failing a single rule, one failing both rules (multiple failing rules for one Pod), each with Deployment/ReplicaSet/Pod-scoped reports.
2. `02-fail-to-pass-via-pod-replacement.yaml` — after fixing the first Deployment's image tag: new Pod, new all-passing report; old Pod/report gone.
3. `03-after-deployment-deletion.yaml` — after deleting the second (multi-rule-failing) Deployment: its reports are gone; includes the orphaned scaled-to-zero ReplicaSet report described above.

## Deferred to hand-crafted fixtures (not practical to force from a live cluster)

Reordered results and malformed/unknown-property results are authored directly as Go test fixtures in `kyverno_v1alpha2_test.go`, based on the real shape captured here.
