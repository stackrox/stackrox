Feature: Map Compliance Operator scheduled scan resources to ACS scan configurations
  As an operator
  I want importer behavior defined by examples
  So implementation can be verified against stable expected outcomes

  Background:
    Given ACS endpoint and token preflight succeeded
    And the importer can read compliance.openshift.io resources

  @mapping @name
  Scenario: Use ScanSettingBinding name as scanName
    Given a ScanSettingBinding "cis-weekly" in namespace "openshift-compliance"
    And the binding references ScanSetting "default-auto-apply"
    And the binding references profiles:
      | name                | kind            |
      | ocp4-cis-node       | Profile         |
      | ocp4-cis-master     | Profile         |
      | my-tailored-profile | TailoredProfile |
    When the importer builds the ACS payload
    Then payload.scanName MUST equal "cis-weekly"                     # IMP-MAP-001
    And payload.scanConfig.profiles MUST equal:
      | value               |
      | my-tailored-profile |
      | ocp4-cis-master     |
      | ocp4-cis-node       |                                    # sorted, deduped

  @mapping @profiles
  Scenario: Default missing profile kind to Profile
    Given a ScanSettingBinding profile reference "custom-x" with no kind
    When the importer resolves profile references
    Then the profile reference kind MUST be treated as "Profile"       # IMP-MAP-002
    And the resulting ACS profile name list MUST include "custom-x"

  @mapping @schedule
  Scenario: Convert ScanSetting schedule into ACS schedule
    Given ScanSetting "daily-scan" has complianceSuiteSettings.schedule "0 0 * * *"
    And ScanSettingBinding "daily-cis" references "daily-scan"
    When the importer maps schedule fields
    Then payload.scanConfig.oneTimeScan MUST be false                  # IMP-MAP-003
    And payload.scanConfig.scanSchedule MUST be present                # IMP-MAP-004

  @mapping @description
  Scenario: Build helpful description without ownership marker
    Given ScanSettingBinding "cis-weekly" in namespace "openshift-compliance"
    When the importer builds payload description
    Then payload.scanConfig.description MUST contain "Imported from CO ScanSettingBinding openshift-compliance/cis-weekly"   # IMP-MAP-005
    And payload.scanConfig.description SHOULD include settings reference context                                              # IMP-MAP-006

  @mapping @clusters
  Scenario: Use single destination ACS cluster ID
    Given importer flag --acs-cluster-id is "cluster-a"
    When the importer builds the destination payload
    Then payload.clusters MUST equal:
      | value     |
      | cluster-a |                                                    # IMP-MAP-007

  @validation @mapping
  Scenario: Missing ScanSetting reference fails only that binding
    Given ScanSettingBinding "broken-binding" references ScanSetting "does-not-exist"
    When the importer processes all discovered bindings
    Then "broken-binding" MUST be marked failed                         # IMP-MAP-008
    And problems list MUST include an entry for "broken-binding"        # IMP-MAP-009
    And that problem entry MUST include a fix hint                       # IMP-MAP-010
    And other valid bindings MUST still be processed                     # IMP-MAP-011

  @mapping @schedule @problems
  Scenario: Invalid schedule is collected as problem and skipped
    Given ScanSetting "bad-schedule" has complianceSuiteSettings.schedule "every day at noon"
    And ScanSettingBinding "broken-schedule-binding" references "bad-schedule"
    When the importer maps schedule fields
    Then "broken-schedule-binding" MUST be skipped                       # IMP-MAP-012
    And problems list MUST include category "mapping"                    # IMP-MAP-013
    And problem description MUST mention schedule conversion failed       # IMP-MAP-014
    And problem fix hint MUST suggest using a valid cron expression      # IMP-MAP-015
