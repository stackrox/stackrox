package deployments

import (
	"github.com/stackrox/rox/central/views/common"
)

// implements DeploymentCore interface
type deploymentResponse struct {
	common.ResourceCountByImageCVESeverity
	DeploymentID string `db:"deployment_id"`
}

func (i *deploymentResponse) GetDeploymentID() string {
	return i.DeploymentID
}

func (i *deploymentResponse) GetDeploymentCVEsBySeverity() common.ResourceCountByCVESeverity {
	return &i.ResourceCountByImageCVESeverity
}
