package check414

import (
	"os"
	"regexp"

	"github.com/stackrox/rox/central/compliance/framework"
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
		[]string{"Deployments", "Policies"},
		checkNIST414)
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
	deployments := ctx.Data().Deployments()
	for _, deployment := range deployments {
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
						// since this control is clusterkind, returning on first failed condition
						// not all the evidence is recorded
						framework.Failf(ctx, "Deployment has secret file in %d mode instead of 0600", perm)
						return
					}
				}
			}
		}
	}
	framework.Pass(ctx, "Deployment is not using any secret volume mounts")
}

func checkSecretsInEnv(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, policy := range policies {
		matchSecret, err := regexp.MatchString("(?i)secret", policy.GetFields().GetEnv().GetKey())
		if err != nil {
			log.Error(err)
		}
		enabled := false
		if matchSecret && err == nil &&
			policy.GetFields().GetEnv().GetValue() != "" && !policy.GetDisabled() {
			enabled = true
		}

		enforced := false
		if (matchSecret) && err == nil &&
			policy.GetFields().GetEnv().GetValue() != "" && !policy.GetDisabled() &&
			len(policy.GetEnforcementActions()) != 0 {
			enforced = true
		}

		if enabled && enforced {
			framework.Pass(ctx, "Detecting secrets in env is enabled and enforced")
			return
		}
		if enabled && !enforced {
			framework.Fail(ctx, "Detecting secrets in env is enabled and not enforced")
			return
		}
	}
	framework.Fail(ctx, "No policy to detect secrets in env")
}
