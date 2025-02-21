//go:build sql_integration

package test

import (
        "context"
        "testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
        plopStore "github.com/stackrox/rox/central/processlisteningonport/store"
        postgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
        "github.com/stackrox/rox/generated/storage"
        "github.com/stackrox/rox/pkg/fixtures"
        "github.com/stackrox/rox/pkg/postgres/pgtest"
        "github.com/stackrox/rox/pkg/sac"
        "github.com/stackrox/rox/pkg/search"
        "github.com/stretchr/testify/suite"
)

func TestUUID(t *testing.T) {
        suite.Run(t, new(UUIDTestSuite))
}

type UUIDTestSuite struct {
        suite.Suite
        store              plopStore.Store

        postgres *pgtest.TestPostgres
	ctx      context.Context
}

func (s *UUIDTestSuite) SetupTest() {                                             
        s.postgres = pgtest.ForT(s.T())
        s.store = postgresStore.NewFullStore(s.postgres.DB)              
	s.ctx = sac.WithAllAccess(context.Background())
}
                                                                                    
func (s *UUIDTestSuite) TearDownTest() {
        s.postgres.Teardown(s.T())   
}       

func (s *UUIDTestSuite) TestRemovePLOPsWithoutPodUID() {
        plops := []*storage.ProcessListeningOnPortStorage{
                fixtures.GetPlopStorage1(),
                fixtures.GetPlopStorage2(),
                fixtures.GetPlopStorage3(),
                fixtures.GetPlopStorage4(),
                fixtures.GetPlopStorage5(),
                fixtures.GetPlopStorage6(),
        }

        err := s.store.UpsertMany(s.ctx, plops)
        s.NoError(err)
        plopCount, err := s.store.Count(s.ctx, search.EmptyQuery())
        s.NoError(err)
        s.Equal(len(plops), plopCount)

	for _, testCase := range []struct {
		desc            string
		q               *v1.Query
		expectedResults []*storage.ProcessListeningOnPortStorage
		expectErr       bool
	}{
		{
			desc:            "null",
			q:               search.NewQueryBuilder().AddNullField(search.PodUID).ProtoQuery(),
			expectedResults: []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage1(), fixtures.GetPlopStorage2(), fixtures.GetPlopStorage3()},
		},
		{
			desc:            "wildcard",
			q:               search.NewQueryBuilder().AddStrings(search.PodUID, "*").ProtoQuery(),
			expectedResults: []*storage.ProcessListeningOnPortStorage{fixtures.GetPlopStorage4(), fixtures.GetPlopStorage5(), fixtures.GetPlopStorage6()},
		},
		{
			desc:            "empty",
			q:               search.NewQueryBuilder().AddExactMatches(search.PodUID, "").ProtoQuery(),
			expectedResults: []*storage.ProcessListeningOnPortStorage{},
			expectErr:       true,
		},
	} {
		s.Run(testCase.desc, func() {
			results, err := s.store.Search(ctx, testCase.q)
			if testCase.expectErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)

			actualIDs := make([]string, 0, len(results))
			for _, res := range results {
				actualIDs = append(actualIDs, res.ID)
			}

			expectedIDs := make([]string, 0, len(testCase.expectedResults))
			for _, s := range testCase.expectedResults {
				expectedIDs = append(expectedIDs, s.Id)
			}
			s.ElementsMatch(actualIDs, expectedIDs)
		})
	}
}
