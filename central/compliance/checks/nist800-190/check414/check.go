package check414

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const (
	standardID = "NIST_800_190:4_1_4"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Deployments", "Policies"},
			InterpretationText: interpretationText,
		},
		checkNIST414)
}

// This is a partial check. We still need to do,
// * Check if they integrate with vault or such
// * Scan the image for strings that look like keys
// This check only ensures that the secret mounts have
// 0600 permission bits on them.
func checkNIST414(ctx framework.ComplianceContext) {
	common.CheckSecretFilePerms(ctx)
	common.CheckSecretsInEnv(ctx)
}
