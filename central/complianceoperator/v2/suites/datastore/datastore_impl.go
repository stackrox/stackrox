package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/suites/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	store postgres.Store
}

// GetSuite returns the suite with the name
func (d *datastoreImpl) GetSuite(ctx context.Context, id string) (*storage.ComplianceOperatorSuite, bool, error) {
	return d.store.Get(ctx, id)
}

// UpsertSuite adds the suite to the database
func (d *datastoreImpl) UpsertSuite(ctx context.Context, suite *storage.ComplianceOperatorSuite) error {
	return d.store.Upsert(ctx, suite)
}

// UpsertSuites adds the suites to the database
func (d *datastoreImpl) UpsertSuites(ctx context.Context, suites []*storage.ComplianceOperatorSuite) error {
	return d.store.UpsertMany(ctx, suites)
}

// DeleteSuite removes a suite from the database
func (d *datastoreImpl) DeleteSuite(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

// GetSuitesByCluster retrieve suites by the cluster
func (d *datastoreImpl) GetSuitesByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorSuite, error) {
	return d.store.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}
