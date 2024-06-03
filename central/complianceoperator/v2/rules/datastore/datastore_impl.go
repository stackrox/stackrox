package datastore

import (
	"context"

	ruleStore "github.com/stackrox/rox/central/complianceoperator/v2/rules/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

type datastoreImpl struct {
	store ruleStore.Store
	db    postgres.DB
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

// DeleteRulesByCluster delete rule by cluster id
func (d *datastoreImpl) DeleteRulesByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	_, err := d.store.DeleteByQuery(ctx, query)
	if err != nil {
		return err
	}
	return nil
}

// ControlResult represents a result of a control.
type ControlResult struct {
	Control  string `db:"compliance_control"`
	Standard string `db:"compliance_standard"`
	RuleName string `db:"compliance_rule_name"`
}

// GetControlsByRulesAndBenchmarks returns controls by a list of rule names group by control, standard and rule name.
func (d *datastoreImpl) GetControlsByRulesAndBenchmarks(ctx context.Context, ruleNames []string, benchmarks []string) ([]*ControlResult, error) {
	builder := search.NewQueryBuilder()
	builder.AddSelectFields(
		search.NewQuerySelect(search.ComplianceOperatorControl),
		search.NewQuerySelect(search.ComplianceOperatorStandard),
		search.NewQuerySelect(search.ComplianceOperatorRuleName),
	)

	builder.AddExactMatches(search.ComplianceOperatorRuleName, ruleNames...)
	builder.AddExactMatches(search.ComplianceOperatorStandard, benchmarks...)

	// Add a group by clause to group the rule names by name, control and standard to reduce the result set.
	builder.AddGroupBy(
		search.ComplianceOperatorRuleName,
		search.ComplianceOperatorControl,
		search.ComplianceOperatorStandard,
	)

	query := builder.ProtoQuery()
	results, err := pgSearch.RunSelectRequestForSchema[ControlResult](ctx, d.db, postgresSchema.ComplianceOperatorRuleV2Schema, query)
	if err != nil {
		return nil, err
	}

	return results, nil
}
