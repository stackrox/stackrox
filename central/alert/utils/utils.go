package utils

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// GetEntityType returns the type of entity for which the alert was raised
func GetEntityType(alert *storage.Alert) storage.Alert_EntityType {
	if alert == nil || alert.GetEntity() == nil {
		return storage.Alert_UNSET
	}
	switch alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		return storage.Alert_DEPLOYMENT
	case *storage.Alert_Image:
		return storage.Alert_CONTAINER_IMAGE
	case *storage.Alert_Resource_:
		return storage.Alert_RESOURCE
	}
	return storage.Alert_UNSET
}

func ApplyDefaultState(q *v1.Query) *v1.Query {
	var querySpecifiesStateField bool
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if matchFieldQuery.MatchFieldQuery.GetField() == search.ViolationState.String() {
			querySpecifiesStateField = true
		}
	})

	// By default, set stale to false.
	if !querySpecifiesStateField {
		cq := search.ConjunctionQuery(q, search.NewQueryBuilder().AddExactMatches(
			search.ViolationState,
			storage.ViolationState_ACTIVE.String(),
			storage.ViolationState_ATTEMPTED.String()).ProtoQuery())
		cq.Pagination = q.GetPagination()
		return cq
	}
	return q
}
