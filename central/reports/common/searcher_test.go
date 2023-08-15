package common

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestReplaceSearchBySuccess(t *testing.T) {
	testCases := []struct {
		name      string
		query     *v1.Query
		expectedQ *v1.Query
	}{
		{
			name:      "Empty query",
			query:     search.EmptyQuery(),
			expectedQ: search.EmptyQuery(),
		},
		{
			name:      "Match none query",
			query:     search.MatchNoneQuery(),
			expectedQ: search.MatchNoneQuery(),
		},
		{
			name: "Simple query without Report state field",
			query: search.NewQueryBuilder().AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).
				ProtoQuery(),
			expectedQ: search.NewQueryBuilder().AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).
				ProtoQuery(),
		},
		{
			name: "Simple query without Report state = SUCCESS",
			query: search.NewQueryBuilder().
				AddExactMatches(search.ReportState, storage.ReportStatus_PREPARING.String(), storage.ReportStatus_WAITING.String()).
				ProtoQuery(),
			expectedQ: search.NewQueryBuilder().
				AddExactMatches(search.ReportState, storage.ReportStatus_PREPARING.String(), storage.ReportStatus_WAITING.String()).
				ProtoQuery(),
		},
		{
			name: "Complex query without Report state field",
			query: search.ConjunctionQuery(
				search.NewQueryBuilder().
					AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).ProtoQuery(),
				search.NewQueryBuilder().
					AddExactMatches(search.ReportNotificationMethod, storage.ReportStatus_DOWNLOAD.String()).ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, "config1", "config2").ProtoQuery(),
			),
			expectedQ: search.ConjunctionQuery(
				search.NewQueryBuilder().
					AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).ProtoQuery(),
				search.NewQueryBuilder().
					AddExactMatches(search.ReportNotificationMethod, storage.ReportStatus_DOWNLOAD.String()).ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, "config1", "config2").ProtoQuery(),
			),
		},
		{
			name: "Complex query without Report state = SUCCESS",
			query: search.ConjunctionQuery(
				search.NewQueryBuilder().
					AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).ProtoQuery(),
				search.NewQueryBuilder().
					AddExactMatches(search.ReportNotificationMethod, storage.ReportStatus_DOWNLOAD.String()).ProtoQuery(),
				search.NewQueryBuilder().
					AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
					ProtoQuery(),
			),
			expectedQ: search.ConjunctionQuery(
				search.NewQueryBuilder().
					AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).ProtoQuery(),
				search.NewQueryBuilder().
					AddExactMatches(search.ReportNotificationMethod, storage.ReportStatus_DOWNLOAD.String()).ProtoQuery(),
				search.NewQueryBuilder().
					AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
					ProtoQuery(),
			),
		},
		{
			name: "Simple query with Report state = SUCCESS",
			query: search.NewQueryBuilder().
				AddStrings(search.ReportState, storage.ReportStatus_PREPARING.String(), "SUCCESS").ProtoQuery(),
			expectedQ: search.DisjunctionQuery(
				search.NewQueryBuilder().
					AddStrings(search.ReportState, storage.ReportStatus_PREPARING.String()).
					ProtoQuery(),
				search.NewQueryBuilder().
					AddExactMatches(search.ReportState, storage.ReportStatus_GENERATED.String(), storage.ReportStatus_DELIVERED.String()).
					ProtoQuery(),
			),
		},
		{
			name: "Complex query with Report state = SUCCESS",
			query: search.ConjunctionQuery(
				search.NewQueryBuilder().
					AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).ProtoQuery(),
				search.NewQueryBuilder().
					AddExactMatches(search.ReportNotificationMethod, storage.ReportStatus_DOWNLOAD.String()).ProtoQuery(),
				search.NewQueryBuilder().
					AddExactMatches(search.ReportState, storage.ReportStatus_PREPARING.String(), "SUCCESS").
					ProtoQuery(),
			),
			expectedQ: search.ConjunctionQuery(
				search.NewQueryBuilder().
					AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).ProtoQuery(),
				search.NewQueryBuilder().
					AddExactMatches(search.ReportNotificationMethod, storage.ReportStatus_DOWNLOAD.String()).ProtoQuery(),
				search.DisjunctionQuery(
					search.NewQueryBuilder().
						AddExactMatches(search.ReportState, storage.ReportStatus_PREPARING.String()).
						ProtoQuery(),
					search.NewQueryBuilder().
						AddExactMatches(search.ReportState, storage.ReportStatus_GENERATED.String(), storage.ReportStatus_DELIVERED.String()).
						ProtoQuery(),
				),
			),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			replaceSearchBySuccess(tc.query)
			assert.Equal(t, tc.expectedQ, tc.query)
		})
	}
}
