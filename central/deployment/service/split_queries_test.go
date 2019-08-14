package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestUsesConvertedQueryCorrectly(t *testing.T) {
	t.Parallel()

	// create a service instance with mocked risk and deployment data.
	mockCtrl := gomock.NewController(t)
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(mockCtrl)
	mockDeploymentDatastore := mocks.NewMockDataStore(mockCtrl)

	// Fill the ranker with fake deployment risk scores.
	ranker := ranking.NewRanker()
	ranker.Add("dep1", 1.0)
	ranker.Add("dep3", 3.0)
	ranker.Add("dep4", 2.0)

	// Fake risk results.
	fakeRiskResults := []*storage.Risk{
		{
			Subject: &storage.RiskSubject{
				Id:   "dep3",
				Type: storage.RiskSubjectType_DEPLOYMENT,
			},
		},
		{
			Subject: &storage.RiskSubject{
				Id:   "dep4",
				Type: storage.RiskSubjectType_DEPLOYMENT,
			},
		},
		{
			Subject: &storage.RiskSubject{
				Id:   "dep1",
				Type: storage.RiskSubjectType_DEPLOYMENT,
			},
		},
	}

	// Expect the given risk query and return the fake results.
	expectedRiskQuery := func() *v1.Query {
		query := search.NewQueryBuilder().
			AddStrings(search.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).
			ProtoQuery()
		query.Pagination = &v1.QueryPagination{
			SortOptions: []*v1.QuerySortOption{
				{
					Field:    search.RiskScore.String(),
					Reversed: false,
				},
			},
		}
		return query
	}()
	mockRiskDatastore.EXPECT().SearchRawRisks(gomock.Any(), expectedRiskQuery).Return(fakeRiskResults, nil)

	// Fake deployment data, we want 4 deployments even though only 3 have risk.
	fakeDeployments := []*storage.ListDeployment{
		{
			Id: "dep1",
		},
		{
			Id: "dep2",
		},
		{
			Id: "dep3",
		},
		{
			Id: "dep4",
		},
	}
	// Expect the given deployment query and return the fake results.
	expectedDeploymentQuery := func() *v1.Query {
		query := search.NewQueryBuilder().
			AddStrings(search.DeploymentName, "deployment").
			ProtoQuery()
		return query
	}()
	mockDeploymentDatastore.EXPECT().SearchListDeployments(gomock.Any(), expectedDeploymentQuery).Return(fakeDeployments, nil)

	// Expected sorted result should be the three deployments with risk score, followed by the one without.
	expectedDeployments := []*storage.ListDeployment{
		// fakeDeployments[2], // risk score 3.0, excluded by paging.
		fakeDeployments[3], // risk score 2.0
		fakeDeployments[0], // risk score 1.0
		// fakeDeployments[1], // No risk score, excluded by paging.
	}

	// Input to the service, whick should be converted into the above deployment and risk queries.
	inputQuery := func() *v1.Query {
		query := search.NewQueryBuilder().
			AddStrings(search.DeploymentName, "deployment").
			ProtoQuery()
		query.Pagination = &v1.QueryPagination{
			Limit:  2,
			Offset: 1,
			SortOptions: []*v1.QuerySortOption{
				{
					Field:    search.Priority.String(),
					Reversed: true,
				},
			},
		}
		return query
	}()

	actualDeployments, err := newSplitQueryExecutor(inputQuery, ranker, mockDeploymentDatastore, mockRiskDatastore).getListDeployments(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, expectedDeployments, actualDeployments)
}
