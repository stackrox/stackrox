package datastore

import (
	"context"
	"fmt"
	"log"

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
	ComplianceControl string `db:"compliance_control"`
}

func (d datastoreImpl) GetControlByRuleId(ctx context.Context, ruleName string) {
	builder := search.NewQueryBuilder()
	builder.AddSelectFields().
		AddExactMatches(search.ComplianceOperatorRuleName, ruleName)
	builder.AddSelectFields(search.NewQuerySelect(search.ComplianceOperatorControl))
	query := builder.ProtoQuery()

	results, err := pgSearch.RunSelectRequestForSchema[ControlResult](ctx, d.db, postgresSchema.ComplianceOperatorRuleV2Schema, query)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Printf("length %d", len(results))
	for _, result := range results {
		fmt.Printf("RESULT: %+v", result)
	}
	//AddExactMatches()

	//builder.AddSelectFields(search.ComplianceOperatorControl, search.ComplianceOperatorStandard)

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
