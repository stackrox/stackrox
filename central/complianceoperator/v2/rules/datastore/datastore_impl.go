package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/rules/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	store postgres.Store
}

// UpsertRule adds the rule to the database
func (d *datastoreImpl) UpsertRule(ctx context.Context, rule *storage.ComplianceOperatorRuleV2) error {
	return d.store.Upsert(ctx, rule)
}

// DeleteRule removes a rule from the database
func (d *datastoreImpl) DeleteRule(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

// GetRulesByCluster retrieves rules by cluster
func (d *datastoreImpl) GetRulesByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorRuleV2, error) {
	return d.store.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}
