# 05 - Traceability Matrix

Use this matrix to ensure complete implementation coverage.

|Requirement ID|Spec source|Test level|Notes|
|---|---|---|---|
|IMP-CLI-001..026|`01-cli-and-config-contract.md`|Unit + integration|CLI parsing, preflight, token/basic auth modes, create-only report + problems list|
|IMP-MAP-001..015|`02-co-to-acs-mapping.feature`|Unit + integration|Mapping, schedule handling, skip+problem behavior|
|IMP-IDEM-001..007|`03-idempotency-dry-run-retries.feature`|Unit + integration|Create-only idempotency and dry-run reporting|
|IMP-ERR-001..004|`03-idempotency-dry-run-retries.feature`|Unit + integration|Retry classes, skip-on-error behavior, exit code outcomes|
|IMP-ACC-001..013|`04-validation-and-acceptance.md`|Acceptance|Real cluster and ACS verification|

## Coverage rule

For each requirement ID, implementation PR MUST include:

- at least one test case name containing that ID, and
- one short note in PR description summarizing pass evidence for that ID family.
