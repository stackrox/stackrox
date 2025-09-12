package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/suites/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	store postgres.Store
}

// GetSuite returns the suite with the name
func (d *datastoreImpl) GetSuite(ctx context.Context, id string) (*storage.ComplianceOperatorSuiteV2, bool, error) {
	return d.store.Get(ctx, id)
}

// UpsertSuite adds the suite to the database
func (d *datastoreImpl) UpsertSuite(ctx context.Context, suite *storage.ComplianceOperatorSuiteV2) error {
	return d.store.Upsert(ctx, suite)
}

// UpsertSuites adds the suites to the database
func (d *datastoreImpl) UpsertSuites(ctx context.Context, suites []*storage.ComplianceOperatorSuiteV2) error {
	return d.store.UpsertMany(ctx, suites)
}

// DeleteSuite removes a suite from the database
func (d *datastoreImpl) DeleteSuite(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

// GetSuitesByCluster retrieve suites by the cluster
func (d *datastoreImpl) GetSuitesByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorSuiteV2, error) {
	return d.store.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}

// GetSuites retrieves the suites for the query.
func (d *datastoreImpl) GetSuites(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorSuiteV2, error) {
	suites, err := d.store.GetByQuery(ctx, query)

	// Follow the existing all or nothing approach to check status.
	// We must ensure the user has access to all the clusters in a config.
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).AllAllowed(getScopeKeys(suites)) {
		return nil, nil
	}
	return suites, err
}

func getScopeKeys(suites []*storage.ComplianceOperatorSuiteV2) [][]sac.ScopeKey {
	clusterScopeKeys := make([][]sac.ScopeKey, 0, len(suites))
	for _, suite := range suites {
		clusterScopeKeys = append(clusterScopeKeys, []sac.ScopeKey{sac.ClusterScopeKey(suite.GetClusterId())})
	}
	return clusterScopeKeys
}

// DeleteSuiteByCluster removes a suite from the database
func (d *datastoreImpl) DeleteSuitesByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	return d.store.DeleteByQuery(ctx, query)
}
