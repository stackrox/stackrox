package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/rules/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type RuleEvent struct {
	Create func(context.Context, *storage.ComplianceOperatorRuleV2) error
}

var events []RuleEvent

func RegisterRuleEvent(event RuleEvent) {
	events = append(events, event)
}

type datastoreImpl struct {
	store postgres.Store
}

// UpsertRule adds the rule to the database
func (d *datastoreImpl) UpsertRule(ctx context.Context, rule *storage.ComplianceOperatorRuleV2) error {
	if err := d.store.Upsert(ctx, rule); err != nil {
		return err
	}

	//TODO: Creating the link on rule import might be possible but could create unnecessary load on the database
	// Generally, 1000 rules per cluster, 1000 SELECTs on the control table per cluster, 15 clusters = 15000 SELECT statements.
	// + 15000 SELECT statements on benchmark table.

	for _, event := range events {
		if err := event.Create(ctx, rule); err != nil {
			return err
		}
	}
	return nil
}

// DeleteRule removes a rule from the database
func (d *datastoreImpl) DeleteRule(ctx context.Context, id string) error {
	//TODO: delete control link
	return d.store.Delete(ctx, id)
}

// GetRulesByCluster retrieves rules by cluster
func (d *datastoreImpl) GetRulesByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorRuleV2, error) {
	return d.store.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
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
