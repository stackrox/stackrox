package imagecve

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/utils"
)

type imageCVECoreViewImpl struct {
	schema *walker.Schema
	db     *postgres.DB
}

func (v *imageCVECoreViewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if err := validateQuery(q); err != nil {
		return 0, err
	}

	var err error
	var results []*imageCVECoreCount
	results, err = pgSearch.RunSelectRequestForSchema[imageCVECoreCount](ctx, v.db, v.schema, withCountQuery(q))
	if err != nil {
		return 0, err
	}
	if len(results) > 1 {
		utils.Should(errors.Errorf("Unexpected: retrieved multiple rows when only one row is expected for count query %q", q.String()))
	}
	return results[0].CVECount, nil
}

func (v *imageCVECoreViewImpl) Get(ctx context.Context, q *v1.Query) ([]CveCore, error) {
	if err := validateQuery(q); err != nil {
		return nil, err
	}

	results, err := pgSearch.RunSelectRequestForSchema[imageCVECore](ctx, v.db, v.schema, withSelectQuery(q))
	if err != nil {
		return nil, err
	}
	ret := make([]CveCore, 0, len(results))
	for _, r := range results {
		ret = append(ret, r)
	}
	return ret, nil
}

func validateQuery(q *v1.Query) error {
	// We only support a dynamic where clause. CveCore has a pre-defined select and group by. Remember this is a "view".
	if len(q.GetSelects()) > 0 {
		return errors.Errorf("Unexpected select clause in query %q", q.String())
	}
	if q.GetGroupBy() != nil {
		return errors.Errorf("Unexpected group by clause in query %q", q.String())
	}
	return nil
}

func withSelectQuery(q *v1.Query) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = []*v1.QueryField{
		{
			Field: search.CVE.String(),
		},
		{
			Field:         search.CVSS.String(),
			AggregateFunc: pgSearch.MaxAggrFunc.String(),
		},
		{
			Field:         search.ImageSHA.String(),
			AggregateFunc: pgSearch.CountAggrFunc.String(),
		},
		{
			Field:         search.CVECreatedTime.String(),
			AggregateFunc: pgSearch.MinAggrFunc.String(),
		},
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}
	return cloned
}

func withCountQuery(q *v1.Query) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = []*v1.QueryField{
		{
			Field:         search.CVE.String(),
			AggregateFunc: pgSearch.CountAggrFunc.String(),
			Distinct:      true,
		},
	}
	return cloned
}
