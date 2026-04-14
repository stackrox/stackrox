//go:build sql_integration

package test

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	testStore "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/testuuidkey/postgres"
	"github.com/stretchr/testify/suite"
)

func TestUUID(t *testing.T) {
	suite.Run(t, new(UUIDTestSuite))
}

type UUIDTestSuite struct {
	suite.Suite
	store testStore.Store

	postgres *pgtest.TestPostgres
	ctx      context.Context
}

func (s *UUIDTestSuite) SetupTest() {
	s.postgres = pgtest.ForT(s.T())
	s.store = testStore.New(s.postgres.DB)
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *UUIDTestSuite) TestNullableUUIDQueries() {
	// 3 objects without OptionalUuid, 3 with it set
	objs := []*storage.TestSingleUUIDKeyStruct{
		{Key: uuid.NewV4().String(), Name: "no-uuid-1"},
		{Key: uuid.NewV4().String(), Name: "no-uuid-2"},
		{Key: uuid.NewV4().String(), Name: "no-uuid-3"},
		{Key: uuid.NewV4().String(), Name: "with-uuid-1", OptionalUuid: uuid.NewV4().String()},
		{Key: uuid.NewV4().String(), Name: "with-uuid-2", OptionalUuid: uuid.NewV4().String()},
		{Key: uuid.NewV4().String(), Name: "with-uuid-3", OptionalUuid: uuid.NewV4().String()},
	}

	err := s.store.UpsertMany(s.ctx, objs)
	s.NoError(err)
	count, err := s.store.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(len(objs), count)

	for _, testCase := range []struct {
		desc            string
		q               *v1.Query
		expectedResults []*storage.TestSingleUUIDKeyStruct
		expectErr       bool
	}{
		{
			desc:            "null",
			q:               search.NewQueryBuilder().AddNullField(search.TestUUID).ProtoQuery(),
			expectedResults: objs[:3],
		},
		{
			desc:            "wildcard",
			q:               search.NewQueryBuilder().AddStrings(search.TestUUID, "*").ProtoQuery(),
			expectedResults: objs[3:],
		},
		{
			desc:            "empty",
			q:               search.NewQueryBuilder().AddExactMatches(search.TestUUID, "").ProtoQuery(),
			expectedResults: []*storage.TestSingleUUIDKeyStruct{},
			expectErr:       true,
		},
	} {
		s.Run(testCase.desc, func() {
			results, err := s.store.Search(s.ctx, testCase.q)
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
			for _, obj := range testCase.expectedResults {
				expectedIDs = append(expectedIDs, obj.GetKey())
			}
			s.ElementsMatch(actualIDs, expectedIDs)
		})
	}
}
