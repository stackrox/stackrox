//go:build sql_integration

package postgres

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

func (s *TestSingleKeyStructsStoreSuite) TestStoreNilMap() {
	ctx := sac.WithAllAccess(context.Background())

	testSingleKeyStruct := &storage.TestSingleKeyStruct{}
	s.NoError(s.store.Upsert(ctx, testSingleKeyStruct))

	var val string
	row := s.testDB.QueryRow(ctx, "select labels from test_single_key_structs")
	err := row.Scan(&val)
	s.NoError(err)
	s.Equal("{}", val)
}

func (s *TestSingleKeyStructsStoreSuite) TestStoreNilMapUpsertMany() {
	ctx := sac.WithAllAccess(context.Background())

	const batchSize = 10000
	testSingleKeyStructs := make([]*storage.TestSingleKeyStruct, batchSize)
	for i := range testSingleKeyStructs {
		testSingleKeyStructs[i] = &storage.TestSingleKeyStruct{
			Key:  fmt.Sprintf("%d", i),
			Name: fmt.Sprintf("%d", i),
		}
	}
	s.NoError(s.store.UpsertMany(ctx, testSingleKeyStructs))

	var val string
	row := s.testDB.QueryRow(ctx, "select labels from test_single_key_structs limit 1")
	err := row.Scan(&val)
	s.NoError(err)
	s.Equal("{}", val)
}
