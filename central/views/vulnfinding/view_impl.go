package vulnfinding

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

var queryTimeout = env.PostgresVMStatementTimeout.DurationSetting()

type viewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *viewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if err := common.ValidateQuery(q); err != nil {
		return 0, err
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVEID).AggrFunc(aggregatefunc.Count).Proto(),
	}

	result, err := pgSearch.RunSelectOneForSchema[findingCountResponse](queryCtx, v.db, v.schema, cloned)
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}
	return result.Count, nil
}

func (v *viewImpl) Get(ctx context.Context, q *v1.Query) ([]Finding, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.DeploymentID).Proto(),
		search.NewQuerySelect(search.ImageSHA).Proto(),
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.Component).Proto(),
		search.NewQuerySelect(search.ComponentVersion).Proto(),
		search.NewQuerySelect(search.Fixable).Proto(),
		search.NewQuerySelect(search.FixedBy).Proto(),
		search.NewQuerySelect(search.VulnerabilityState).Proto(),
		search.NewQuerySelect(search.Severity).Proto(),
		search.NewQuerySelect(search.CVSS).Proto(),
		search.NewQuerySelect(search.RepositoryCPE).Proto(),
	}

	var ret []Finding
	err := pgSearch.RunSelectRequestForSchemaFn[findingResponse](queryCtx, v.db, v.schema, cloned, func(r *findingResponse) error {
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}
