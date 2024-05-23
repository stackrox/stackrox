package datastore

import (
	"context"

	benchmarkPostgres "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	store benchmarkPostgres.Store
	db    postgres.DB
}

type ControlResult struct {
	Control  string `db:"compliance_control"`
	RuleId   string `db:"compliance_rule_id"`
	Standard string `db:"compliance_standard"`
	RuleName string `db:"compliance_rule_name"`
}

func (d datastoreImpl) GetControlByRuleName(ctx context.Context, ruleNames []string) ([]*ControlResult, error) {
	builder := search.NewQueryBuilder()
	builder.AddSelectFields(
		// TODO: throws error to select ID
		// field name:"compliance rule id"  in select portion of query does not exist in table compliance_operator_rule_v2 or connected tables
		//search.NewQuerySelect(search.ComplianceOperatorRuleId),
		search.NewQuerySelect(search.ComplianceOperatorControl),
		search.NewQuerySelect(search.ComplianceOperatorStandard),
		search.NewQuerySelect(search.ComplianceOperatorRuleName),
	)

	builder.AddExactMatches(search.ComplianceOperatorRuleName, ruleNames...)

	query := builder.ProtoQuery()
	results, err := pgSearch.RunSelectRequestForSchema[ControlResult](ctx, d.db, postgresSchema.ComplianceOperatorRuleV2Schema, query)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (d datastoreImpl) GetBenchmark(ctx context.Context, id string) (*storage.ComplianceOperatorBenchmarkV2, bool, error) {
	return d.store.Get(ctx, id)
}

func (d datastoreImpl) UpsertBenchmark(ctx context.Context, result *storage.ComplianceOperatorBenchmarkV2) error {
	return d.store.Upsert(ctx, result)
}

func (d datastoreImpl) DeleteBenchmark(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}
