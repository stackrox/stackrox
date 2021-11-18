package getters

import (
	"context"
	"strconv"

	"github.com/stackrox/rox/central/alert/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var (
	policyNameField = mappings.OptionsMap.MustGet(search.PolicyName.String())
	severityField   = mappings.OptionsMap.MustGet(search.Severity.String())
)

// MockAlertsSearcher is a mock AlertsSearcher.
type MockAlertsSearcher struct {
	Alerts []*storage.ListAlert
}

// Search supports a limited set of request parameters.
// It only needs to be as specific as the production code.
func (m MockAlertsSearcher) Search(ctx context.Context, q *v1.Query) (results []search.Result, err error) {
	state := storage.ViolationState_ACTIVE.String()
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		mfQ, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok && mfQ.MatchFieldQuery.GetField() == search.ViolationState.String() {
			state = mfQ.MatchFieldQuery.GetValue()
		}
	})

	for _, a := range m.Alerts {
		if a.GetState().String() == state {
			results = append(results, search.Result{
				ID: a.GetId(),
				Matches: map[string][]string{
					policyNameField.FieldPath: {a.GetPolicy().GetName()},
					severityField.FieldPath:   {strconv.Itoa(int(a.GetPolicy().GetSeverity()))},
				},
			})
		}
	}
	return
}
