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
    Given ScanSetting "daily-scan" has schedule "0 0 * * *"
    And ScanSettingBinding "daily-cis" references "daily-scan"
    When the importer maps schedule fields
    Then payload.scanConfig.oneTimeScan MUST be false                  # IMP-MAP-003
    And payload.scanConfig.scanSchedule MUST be present                # IMP-MAP-004

  @mapping @schedule @wire-format
  Scenario Outline: Schedule JSON wire format matches ACS API proto
    Given ScanSetting "sched" has schedule "<cron>"
    And ScanSettingBinding "binding" references "sched"
    When the importer builds the ACS payload and serializes it to JSON
    Then the JSON scanSchedule object MUST contain only fields defined in proto/api/v2/common.proto Schedule:
      intervalType, hour, minute, daysOfWeek, daysOfMonth               # IMP-MAP-004a
    And the JSON scanSchedule.intervalType MUST be "<intervalType>"
    And for WEEKLY: scanSchedule.daysOfWeek.days MUST be present        # IMP-MAP-004b
    And for MONTHLY: scanSchedule.daysOfMonth.days MUST be present      # IMP-MAP-004c
    And the full payload JSON field names MUST match ComplianceScanConfiguration proto:
      id, scanName, scanConfig, clusters                                # IMP-MAP-004d

    Examples:
      | cron         | intervalType |
      | 0 2 * * *    | DAILY        |
      | 0 2 * * 0    | WEEKLY       |
      | 0 2 1 * *    | MONTHLY      |

  @mapping @description
  Scenario: Build helpful description without ownership marker
    Given ScanSettingBinding "cis-weekly" in namespace "openshift-compliance"
    When the importer builds payload description
    Then payload.scanConfig.description MUST contain "Imported from CO ScanSettingBinding openshift-compliance/cis-weekly"   # IMP-MAP-005
    And payload.scanConfig.description SHOULD include settings reference context                                              # IMP-MAP-006

  @mapping @clusters
  Scenario: Auto-discover ACS cluster ID from admission-control ConfigMap
    Given kubecontext "ctx-a" points to a secured cluster
    And ConfigMap "admission-control" in namespace "stackrox" has data key "cluster-id" = "uuid-a"
    When the importer resolves the ACS cluster ID for "ctx-a"
    Then the resolved ACS cluster ID MUST be "uuid-a"                  # IMP-MAP-016

  @mapping @clusters
  Scenario: Fallback to OpenShift ClusterVersion for cluster matching
    Given kubecontext "ctx-b" points to an OpenShift cluster
    And ConfigMap "admission-control" is not readable
    And ClusterVersion "version" has spec.clusterID "ocp-uuid-b"
    And ACS cluster list contains a cluster with providerMetadata.cluster.id "ocp-uuid-b" and ACS ID "acs-uuid-b"
    When the importer resolves the ACS cluster ID for "ctx-b"
    Then the resolved ACS cluster ID MUST be "acs-uuid-b"              # IMP-MAP-017

  @mapping @clusters
  Scenario: Fallback to helm-effective-cluster-name for cluster matching
    Given kubecontext "ctx-c" points to a cluster
    And ConfigMap "admission-control" is not readable
    And ClusterVersion is not available
    And Secret "helm-effective-cluster-name" has data key "cluster-name" = "my-cluster"
    And ACS cluster list contains a cluster named "my-cluster" with ACS ID "acs-uuid-c"
    When the importer resolves the ACS cluster ID for "ctx-c"
    Then the resolved ACS cluster ID MUST be "acs-uuid-c"              # IMP-MAP-018

  @mapping @clusters
  Scenario: All discovery methods fail with detailed per-method errors
    Given kubecontext "ctx-d" points to a cluster
    And ConfigMap "admission-control" is not readable (returns "Unauthorized")
    And ClusterVersion is not available (returns "Unauthorized")
    And Secret "helm-effective-cluster-name" is not readable (returns "Unauthorized")
    When the importer resolves the ACS cluster ID for "ctx-d"
    Then the error MUST list each method's failure reason                  # IMP-MAP-016a

  @mapping @clusters @multicluster
  Scenario: Merge SSBs with same name across clusters
    Given kubecontext "ctx-a" has ScanSettingBinding "cis-weekly" with profiles ["ocp4-cis"] and schedule "0 2 * * 0"
    And kubecontext "ctx-b" has ScanSettingBinding "cis-weekly" with profiles ["ocp4-cis"] and schedule "0 2 * * 0"
    And ctx-a resolves to ACS cluster ID "uuid-a"
    And ctx-b resolves to ACS cluster ID "uuid-b"
    When the importer merges SSBs across clusters
    Then one ACS scan config MUST be created with scanName "cis-weekly" # IMP-MAP-019
    And payload.clusters MUST equal:
      | value  |
      | uuid-a |
      | uuid-b |                                                       # IMP-MAP-021

  @mapping @clusters @multicluster @error
  Scenario: Error when same-name SSBs have mismatched profiles
    Given kubecontext "ctx-a" has ScanSettingBinding "cis-weekly" with profiles ["ocp4-cis"]
    And kubecontext "ctx-b" has ScanSettingBinding "cis-weekly" with profiles ["ocp4-cis", "ocp4-moderate"]
    When the importer merges SSBs across clusters
    Then "cis-weekly" MUST be marked failed                            # IMP-MAP-020
    And problems list MUST include category "conflict"
    And problem description MUST mention mismatch across clusters
    And the console MUST print a warning with the conflict reason       # IMP-MAP-020a

  @mapping @clusters @multicluster @error
  Scenario: Error when same-name SSBs have mismatched schedules
    Given kubecontext "ctx-a" has ScanSettingBinding "cis-weekly" with schedule "0 2 * * 0"
    And kubecontext "ctx-b" has ScanSettingBinding "cis-weekly" with schedule "0 3 * * 1"
    When the importer merges SSBs across clusters
    Then "cis-weekly" MUST be marked failed                            # IMP-MAP-020
    And problems list MUST include category "conflict"
    And problem description MUST mention mismatch across clusters
    And the console MUST print a warning with the conflict reason       # IMP-MAP-020a

  @validation @mapping
  Scenario: Missing ScanSetting reference fails only that binding
    Given ScanSettingBinding "broken-binding" references ScanSetting "does-not-exist"
    When the importer processes all discovered bindings
    Then "broken-binding" MUST be marked failed                         # IMP-MAP-008
    And problems list MUST include an entry for "broken-binding"        # IMP-MAP-009
    And that problem entry MUST include a fix hint                       # IMP-MAP-010
    And other valid bindings MUST still be processed                     # IMP-MAP-011

  @mapping @adopt
  Scenario: Adopt SSB after successful ACS scan config creation
    Given ScanSettingBinding "cis-weekly" in namespace "openshift-compliance"
    And the SSB references ScanSetting "my-old-setting"
    And the importer successfully creates ACS scan config "cis-weekly"
    And ACS creates ScanSetting "cis-weekly" on the cluster
    When the importer runs the adoption step
    Then SSB "cis-weekly" settingsRef.name MUST be patched to "cis-weekly"  # IMP-ADOPT-001
    And the importer MUST log an info message about the adoption            # IMP-ADOPT-002

  @mapping @adopt
  Scenario: Skip adoption when SSB already references the correct ScanSetting
    Given ScanSettingBinding "cis-weekly" in namespace "openshift-compliance"
    And the SSB references ScanSetting "cis-weekly"
    And the importer successfully creates ACS scan config "cis-weekly"
    When the importer runs the adoption step
    Then SSB "cis-weekly" settingsRef.name MUST NOT be modified             # IMP-ADOPT-003

  @mapping @adopt @timeout
  Scenario: Adoption warns on timeout waiting for ScanSetting
    Given ScanSettingBinding "cis-weekly" in namespace "openshift-compliance"
    And the SSB references ScanSetting "my-old-setting"
    And the importer successfully creates ACS scan config "cis-weekly"
    And ACS has NOT yet created ScanSetting "cis-weekly" on the cluster
    When the adoption poll times out
    Then the importer MUST log a warning                                    # IMP-ADOPT-004
    And the SSB MUST NOT be modified                                        # IMP-ADOPT-005
    And the importer MUST NOT exit with an error                            # IMP-ADOPT-006

  @mapping @adopt @multicluster
  Scenario: Adoption patches SSBs independently per cluster
    Given kubecontext "ctx-a" has SSB "cis-weekly" referencing ScanSetting "setting-a"
    And kubecontext "ctx-b" has SSB "cis-weekly" referencing ScanSetting "setting-b"
    And the importer creates one ACS scan config "cis-weekly" for both clusters
    And ACS creates ScanSetting "cis-weekly" on both clusters
    When the importer runs the adoption step
    Then SSB "cis-weekly" on ctx-a MUST be patched to reference "cis-weekly" # IMP-ADOPT-007
    And SSB "cis-weekly" on ctx-b MUST be patched to reference "cis-weekly"  # IMP-ADOPT-007

  @mapping @adopt @multicluster @partial
  Scenario: Partial adoption succeeds when one cluster times out
    Given kubecontext "ctx-a" has SSB "cis-weekly" referencing ScanSetting "setting-a"
    And kubecontext "ctx-b" has SSB "cis-weekly" referencing ScanSetting "setting-b"
    And ACS creates ScanSetting "cis-weekly" on ctx-a but NOT on ctx-b
    When the importer runs the adoption step
    Then SSB "cis-weekly" on ctx-a MUST be patched                          # IMP-ADOPT-008
    And the importer MUST warn about ctx-b timeout                          # IMP-ADOPT-008
    And the importer MUST NOT exit with an error                            # IMP-ADOPT-006

  @mapping @schedule @problems
  Scenario: Invalid schedule is collected as problem and skipped
    Given ScanSetting "bad-schedule" has schedule "every day at noon"
    And ScanSettingBinding "broken-schedule-binding" references "bad-schedule"
    When the importer maps schedule fields
    Then "broken-schedule-binding" MUST be skipped                       # IMP-MAP-012
    And problems list MUST include category "mapping"                    # IMP-MAP-013
    And problem description MUST mention schedule conversion failed       # IMP-MAP-014
    And problem fix hint MUST suggest using a valid cron expression      # IMP-MAP-015
