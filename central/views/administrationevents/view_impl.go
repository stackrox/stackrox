package administrationevents

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var queryTimeout = env.PostgresVMStatementTimeout.DurationSetting()

type eventViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *eventViewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if err := common.ValidateQuery(q); err != nil {
		return 0, err
	}
	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	return pgSearch.RunCountRequestForSchema(queryCtx, v.schema, q, v.db)
}

func (v *eventViewImpl) Get(ctx context.Context, q *v1.Query) ([]*storage.AdministrationEvent, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}
	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	// Read the complete stored event: message, hint and resource name are not
	// search fields, and legacy resource IDs may exist only in the stored payload.
	var events []*storage.AdministrationEvent
	err := pgSearch.RunQueryForSchemaFn(queryCtx, v.schema, q, v.db, func(event *storage.AdministrationEvent) error {
		events = append(events, event)
		return nil
	})
	return events, err
}

// WithWorkloadQuery restricts image events to images in matching deployments.
// Other resource types are unaffected. Pagination belongs to the event query.
func WithWorkloadQuery(q *v1.Query, workloadQuery *v1.Query) (*v1.Query, error) {
	if workloadQuery == nil {
		return q, nil
	}
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}
	if err := common.ValidateQuery(workloadQuery); err != nil {
		return nil, err
	}
	workload := workloadQuery.CloneVT()
	workload.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.ImageID).Proto()}
	workload.Pagination = nil

	// Intentionally use only searchable resource IDs. Older events remain visible
	// unfiltered until normal writes or retention resolve the gap, avoiding legacy
	// event deserialization on every filtered request.
	imageType := search.NewQueryBuilder().AddExactMatches(search.ResourceType, "Image").ProtoQuery()
	images := search.ConjunctionQuery(imageType, search.MatchFieldSubQuery(search.ResourceID, v1.SearchCategory_DEPLOYMENTS, workload))
	otherTypes := search.NewBooleanQuery(nil, &v1.DisjunctionQuery{Queries: []*v1.Query{imageType}})
	query := search.ConjunctionQuery(q.CloneVT(), search.DisjunctionQuery(images, otherTypes))
	query.Pagination = q.GetPagination().CloneVT()
	return query, nil
}
