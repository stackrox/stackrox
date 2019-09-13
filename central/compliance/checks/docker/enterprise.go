package docker

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:8_1_1", "Check if UCP is configured to use external LDAP authentication service"),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:8_1_2", "Check if UCP is used with externally trusted certificate authorities"),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:8_1_3", "Check if UCP is used with client certificate bundles for unprivileged users"),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:8_1_4", "Check if UCP is used with custom RBAC policies"),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:8_1_5", "Check if UCP is used with signed image enforcement enabled"),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:8_1_6", "Check if UCP is used with the Per-User Session Limit set to a value of '3' or lower"),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:8_1_7", "Check if UCP is used with \"Lifetime Minutes\" and \"Renewal Threshold Minutes\" values set to '15' or lower and '0' respectively"),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:8_2_1", "Check if DTR has image vulnerability scan enabled"),
	)
}
