# 05 - Traceability Matrix

Use this matrix to ensure complete implementation coverage.

|Requirement ID|Spec source|Test level|Notes|
|---|---|---|---|
|IMP-CLI-001..027|`01-cli-and-config-contract.md`|Unit + integration|CLI parsing, preflight, auth modes, multi-cluster, --overwrite-existing|
|IMP-MAP-001..021, IMP-MAP-020a|`02-co-to-acs-mapping.feature`|Unit + integration|Mapping, schedule, cluster auto-discovery, SSB merging, merge conflict console output|
|IMP-IDEM-001..009|`03-idempotency-dry-run-retries.feature`|Unit + integration|Idempotency, overwrite mode (PUT), dry-run reporting|
|IMP-ERR-001..004|`03-idempotency-dry-run-retries.feature`|Unit + integration|Retry classes, skip-on-error behavior, exit code outcomes|
|IMP-ACC-001..017|`04-validation-and-acceptance.md`|Acceptance|Real cluster, ACS verification, multi-cluster merge, auto-discovery|
|IMP-IMG-001..006|`07-container-image.md`|Build + smoke|Dockerfile, static binary, multi-arch manifest, image size|

## Coverage rule

For each requirement ID, implementation PR MUST include:

- at least one test case name containing that ID, and
- one short note in PR description summarizing pass evidence for that ID family.
