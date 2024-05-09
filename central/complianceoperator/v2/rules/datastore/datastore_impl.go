package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/rules/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
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

// SearchRules returns the rules for the given query
func (d *datastoreImpl) SearchRules(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorRuleV2, error) {
	return d.store.GetByQuery(ctx, query)
}

// delete rule by cluster id
func (d *datastoreImpl) DeleteRulesByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	_, err := d.store.DeleteByQuery(ctx, query)
	if err != nil {
		return err
	}
	return nil
}
