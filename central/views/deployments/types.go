package deployments

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// DeploymentCore is an interface to get deployment properties.
//
//go:generate mockgen-wrapper
type DeploymentCore interface {
	GetDeploymentID() string
	GetDeploymentCVEsBySeverity() common.ResourceCountByCVESeverity
}

// DeploymentView interface provides functionality to fetch the deployment data
type DeploymentView interface {
	Get(ctx context.Context, q *v1.Query) ([]DeploymentCore, error)
}
