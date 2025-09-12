package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/remediations/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	store postgres.Store
}

func (d datastoreImpl) GetRemediation(ctx context.Context, id string) (*storage.ComplianceOperatorRemediationV2, bool, error) {
	return d.store.Get(ctx, id)
}

func (d datastoreImpl) UpsertRemediation(ctx context.Context, result *storage.ComplianceOperatorRemediationV2) error {
	return d.store.Upsert(ctx, result)
}

func (d datastoreImpl) DeleteRemediation(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

func (d datastoreImpl) GetRemediationsByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorRemediationV2, error) {
	queryBuilder := search.NewQueryBuilder()
	query := queryBuilder.AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	return d.store.GetByQuery(ctx, query)
}

func (d datastoreImpl) DeleteRemediationsByCluster(ctx context.Context, clusterID string) error {
	queryBuilder := search.NewQueryBuilder()
	query := queryBuilder.AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	return d.store.DeleteByQuery(ctx, query)
}

// SearchRemediations returns the remediations for the given query
func (d *datastoreImpl) SearchRemediations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorRemediationV2, error) {
	return d.store.GetByQuery(ctx, query)
}
