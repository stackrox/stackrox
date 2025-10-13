package deployments

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	deploymentView DeploymentView
)

// NewDeploymentView returns the interface DeploymentView
// that provides methods for searching deployments stored in the database.
func NewDeploymentView(db postgres.DB) DeploymentView {
	if !features.FlattenCVEData.Enabled() {
		return nil
	}

	return &deploymentViewImpl{
		db:     db,
		schema: schema.DeploymentsSchema,
	}
}

// Singleton provides the interface to search deployments stored in the database.
func Singleton() DeploymentView {
	once.Do(func() {
		deploymentView = NewDeploymentView(globaldb.GetPostgres())
	})
	return deploymentView
}
