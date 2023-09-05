package compliancemanager

import (
	"context"

	"github.com/pkg/errors"
	integrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type managerImpl struct {
	sensorConnMgr connection.Manager
	integrationDS integrationDS.DataStore
}

// New returns on instance of Manager interface that provides functionality to process compliance requests and forward them to Sensor.
func New(sensorConnMgr connection.Manager, integrationDS integrationDS.DataStore) Manager {
	return &managerImpl{
		sensorConnMgr: sensorConnMgr,
		integrationDS: integrationDS,
	}
}

func (m *managerImpl) Sync(_ context.Context) {
	//TODO: Sync scan configurations with sensor
}

// ProcessComplianceOperatorInfo processes and stores the compliance operator metadata coming from sensor
func (m *managerImpl) ProcessComplianceOperatorInfo(ctx context.Context, complianceIntegration *storage.ComplianceIntegration) error {
	// Check and see if we have this info already.
	existingIntegrations, err := m.integrationDS.GetComplianceIntegrations(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, complianceIntegration.GetClusterId()).ProtoQuery())
	if err != nil {
		return err
	}
	// TODO (ROX-18101):  Shouldn't happen once ROX-18101 is implemented.  Deferring more thorough handling
	// of this condition to that ticket.
	if len(existingIntegrations) > 1 {
		return errors.Errorf("multiple compliance operators for cluster %q exist.", complianceIntegration.GetClusterId())
	}

	// Not found so an add
	if len(existingIntegrations) == 0 {
		_, err := m.integrationDS.AddComplianceIntegration(ctx, complianceIntegration)
		return err
	}

	// An update, so we need the ID from the one that was returned
	id := existingIntegrations[0].GetId()
	complianceIntegration.Id = id

	return m.integrationDS.UpdateComplianceIntegration(ctx, complianceIntegration)
}

func (m *managerImpl) ProcessScanRequest(_ context.Context, _ interface{}) error {
	//TODO:
	// 1. Upsert to database
	// 2. Push request to Sensor
	panic("implement me")
}

func (m *managerImpl) ProcessRescanRequest(_ context.Context, _ interface{}) error {
	//TODO:
	// 1. Validate config exists in database
	// 2. Push request to Sensor
	panic("implement me")
}

func (m *managerImpl) DeleteScan(_ context.Context, _ interface{}) error {
	//TODO:
	// 1. Validate config exists in database
	// 2. Push request to Sensor
	panic("implement me")
}
