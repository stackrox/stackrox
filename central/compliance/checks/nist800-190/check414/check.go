package check414

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
	"github.com/stackrox/stackrox/pkg/logging"
)

const (
	standardID = "NIST_800_190:4_1_4"
)

var (
	log = logging.ModuleForName(standardID).Logger()
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
