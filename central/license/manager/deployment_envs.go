package manager

import (
	"github.com/pkg/errors"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/pkg/set"
)

type deploymentEnvListener struct {
	manager *manager
}

func (l deploymentEnvListener) OnUpdate(clusterID string, deploymentEnvs []string) {
	l.manager.interrupt()
}

func (l deploymentEnvListener) OnClusterMarkedInactive(clusterID string) {
	l.manager.interrupt()
}

func checkDeploymentEnvironmentRestrictions(restr *licenseproto.License_Restrictions, deploymentEnvsByClusterID map[string][]string) error {
	if restr.GetNoDeploymentEnvironmentRestriction() {
		return nil
	}

	if deploymentEnvsByClusterID == nil {
		return errors.New("no deployment environment data available yet (this is a temporary condition)")
	}

	allowedDeploymentEnvs := set.NewStringSet(restr.GetDeploymentEnvironments()...)

	for clusterID, envs := range deploymentEnvsByClusterID {
		for _, env := range envs {
			if !allowedDeploymentEnvs.Contains(env) {
				return errors.Errorf("cluster with ID %s is being used in deployment environment %q, which is not allowed by the license", clusterID, env)
			}
		}
	}

	return nil
}
