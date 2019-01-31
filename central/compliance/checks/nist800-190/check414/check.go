package check414

import (
	"os"
	"regexp"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	standardID = "NIST_800_190:4_1_4"
)

var (
	log = logging.New(standardID)
)

func init() {
	framework.MustRegisterNewCheck(
		standardID,
		framework.ClusterKind,
		[]string{"Policies"},
		func(ctx framework.ComplianceContext) {
			checkNIST414(ctx)
		})
}

// This is a partial check. We still need to do,
// * Check if they integrate with vault or such
// * Scan the image for strings that look like keys
// This check only ensures that the secret mounts have
// 0600 permission bits on them.
func checkNIST414(ctx framework.ComplianceContext) {
	checkSecretFilePerms(ctx)
	checkSecretsInEnv(ctx)
}

func checkSecretFilePerms(ctx framework.ComplianceContext) {
	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		secretFilePath := ""
		for _, container := range deployment.Containers {
			for _, vol := range container.Volumes {
				if vol.Type == "secret" {
					secretFilePath = vol.GetDestination() + vol.Name
					info, err := os.Lstat(secretFilePath)
					if err != nil {
						log.Error(err)
						continue
					}
					perm := info.Mode().Perm()
					if perm != 0600 {
						framework.Failf(ctx, "Deployment has secret file in %d mode instead of 0600", perm)
					} else {
						framework.Pass(ctx, "Deployment is using secrets securely")
					}
				}
			}
		}
	})
}

func checkSecretsInEnv(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, policy := range policies {
		matchUpperCaseSecret, err := regexp.MatchString("SECRET", policy.GetFields().GetEnv().GetKey())
		if err != nil {
			log.Error(err)
		}
		matchLowerCaseSecret, err := regexp.MatchString("secret", policy.GetFields().GetEnv().GetKey())
		if err != nil {
			log.Error(err)
		}
		if (matchUpperCaseSecret || matchLowerCaseSecret) && err == nil &&
			policy.GetFields().GetEnv().GetValue() != "" && !policy.GetDisabled() &&
			len(policy.GetEnforcementActions()) != 0 {
			framework.Pass(ctx, "Detecting Secrets in env is enabled and enforced")
			return
		}
	}
}
