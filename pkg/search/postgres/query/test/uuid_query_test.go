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

// TestLargeUUIDParameterSetUsesANY verifies that queries with more values than
// PostgresParameterThreshold work correctly on UUID-typed columns. The ANY($$)
// path relies on pgx inferring uuid[] from the element types; this test catches
// regressions where the wrong type is inferred and PostgreSQL rejects the query.
func (s *UUIDTestSuite) TestLargeUUIDParameterSetUsesANY() {
	const matchCount = 10
	const totalValues = 101 // > default threshold of 100, triggers ANY path

	// Insert structs with known UUID keys and known optional_uuid values.
	objs := make([]*storage.TestSingleUUIDKeyStruct, matchCount)
	knownKeys := make([]string, matchCount)
	knownOptionalUUIDs := make([]string, matchCount)
	for i := range objs {
		key := uuid.NewV4().String()
		optUUID := uuid.NewV4().String()
		knownKeys[i] = key
		knownOptionalUUIDs[i] = optUUID
		objs[i] = &storage.TestSingleUUIDKeyStruct{
			Key:          key,
			Name:         uuid.NewV4().String(),
			OptionalUuid: optUUID,
		}
	}
	s.Require().NoError(s.store.UpsertMany(s.ctx, objs))

	// Build value lists: known values first, padded with non-matching UUIDs.
	keyValues := make([]string, totalValues)
	copy(keyValues, knownKeys)
	for i := matchCount; i < totalValues; i++ {
		keyValues[i] = uuid.NewV4().String()
	}

	optUUIDValues := make([]string, totalValues)
	copy(optUUIDValues, knownOptionalUUIDs)
	for i := matchCount; i < totalValues; i++ {
		optUUIDValues[i] = uuid.NewV4().String()
	}

	s.Run("uuid primary key column", func() {
		q := search.NewQueryBuilder().AddExactMatches(search.TestKey, keyValues...).ProtoQuery()
		results, err := s.store.Search(s.ctx, q)
		s.Require().NoError(err)
		s.Len(results, matchCount)
	})

	s.Run("uuid non-pk column", func() {
		q := search.NewQueryBuilder().AddExactMatches(search.TestUUID, optUUIDValues...).ProtoQuery()
		results, err := s.store.Search(s.ctx, q)
		s.Require().NoError(err)
		s.Len(results, matchCount)
	})
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
