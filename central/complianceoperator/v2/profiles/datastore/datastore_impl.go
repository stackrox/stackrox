package datastore

import (
	"github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	integrationSAC = sac.ForResource(resources.ComplianceOperator)
)

type datastoreImpl struct {
	storage postgres.Store
}
