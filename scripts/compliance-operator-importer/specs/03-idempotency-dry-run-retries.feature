Feature: Create-only idempotency dry-run behavior and retry policy
  As an operator
  I want safe reruns and predictable failure handling
  So importer usage is low risk in production environments

  Background:
    Given ACS endpoint and token preflight succeeded
    And desired payload for source "openshift-compliance/cis-weekly" is computed

  @idempotency
  Scenario: Create when scanName does not exist
    Given ACS has no scan configuration with scanName "cis-weekly"
    When importer executes in apply mode
    Then importer MUST send POST /v2/compliance/scan/configurations     # IMP-IDEM-001
    And action MUST be "create"

  @idempotency
  Scenario: Skip when scanName already exists (default mode)
    Given ACS has scan configuration with scanName "cis-weekly"
    And --overwrite-existing is false
    When importer executes in apply mode
    Then importer MUST NOT send PUT                                      # IMP-IDEM-003
    And action MUST be "skip"
    And reason MUST include "already exists"
    And problems list MUST include conflict category                     # IMP-IDEM-002

  @idempotency @overwrite
  Scenario: Update when scanName already exists and --overwrite-existing is true
    Given ACS has scan configuration with scanName "cis-weekly" and id "existing-id"
    And --overwrite-existing is true
    When importer executes in apply mode
    Then importer MUST send PUT /v2/compliance/scan/configurations/existing-id  # IMP-IDEM-008
    And action MUST be "update"

  @idempotency @overwrite
  Scenario: Create when scanName does not exist and --overwrite-existing is true
    Given ACS has no scan configuration with scanName "new-scan"
    And --overwrite-existing is true
    When importer executes in apply mode
    Then importer MUST send POST /v2/compliance/scan/configurations     # IMP-IDEM-009
    And action MUST be "create"

  @dryrun
  Scenario: Dry-run performs no writes
    Given importer is started with --dry-run
    And at least one action would be create in apply mode
    When importer completes
    Then importer MUST NOT send POST                                     # IMP-IDEM-004
    And importer MUST NOT send PUT                                       # IMP-IDEM-005
    And planned actions MUST be included in report                       # IMP-IDEM-006
    And problems list MUST still be populated for problematic resources  # IMP-IDEM-007

  @retry @transient
  Scenario Outline: Retry transient ACS write failures
    Given an ACS create operation returns HTTP <status> for first 2 attempts
    And the 3rd attempt succeeds
    When importer executes in apply mode
    Then operation MUST be retried with backoff                          # IMP-ERR-001
    And total attempts MUST be 3

    Examples:
      | status |
      | 429    |
      | 502    |
      | 503    |
      | 504    |

  @retry @nontransient
  Scenario Outline: Do not retry non-transient errors
    Given an ACS create operation returns HTTP <status>
    When importer executes in apply mode
    Then operation MUST NOT be retried                                   # IMP-ERR-002
    And the item MUST be skipped and recorded as a problem               # IMP-ERR-004

    Examples:
      | status |
      | 400    |
      | 401    |
      | 403    |
      | 404    |

  @exitcodes
  Scenario Outline: Exit code reflects outcome category
    Given importer run ends with outcome "<outcome>"
    Then process exit code MUST be <code>                                # IMP-ERR-003

    Examples:
      | outcome                   | code |
      | all successful            | 0    |
      | fatal preflight failure   | 1    |
      | partial binding failures  | 2    |
