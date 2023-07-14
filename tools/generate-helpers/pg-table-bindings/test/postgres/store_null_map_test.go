//go:build sql_integration

package postgres

import (
	"context"

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
	s.Equal("null", val)
}
