//go:build sql_integration

package postgres

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
)

func (s *NamespacesStoreSuite) TestReadsObserveOtherStoreWrites() {
	ctx := sac.WithAllAccess(context.Background())
	writer := New(s.testDB.DB)
	namespace := &storage.NamespaceMetadata{
		Id: uuid.NewV4().String(), Name: "shared-namespace", ClusterId: uuid.NewV4().String(),
		Labels: map[string]string{"team": "initial"},
	}
	_, found, err := s.store.Get(ctx, namespace.GetId())
	s.Require().NoError(err)
	s.Require().False(found)

	s.Require().NoError(writer.Upsert(ctx, namespace))
	result, found, err := s.store.Get(ctx, namespace.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Equal(namespace.GetLabels(), result.GetLabels())

	namespace.Labels = map[string]string{"team": "updated"}
	s.Require().NoError(writer.Upsert(ctx, namespace))
	results, missing, err := s.store.GetMany(ctx, []string{namespace.GetId()})
	s.Require().NoError(err)
	s.Require().Empty(missing)
	s.Require().Len(results, 1)
	s.Equal(namespace.GetLabels(), results[0].GetLabels())

	s.Require().NoError(writer.Delete(ctx, namespace.GetId()))
	_, found, err = s.store.Get(ctx, namespace.GetId())
	s.Require().NoError(err)
	s.False(found)
	results, missing, err = s.store.GetMany(ctx, []string{namespace.GetId()})
	s.Require().NoError(err)
	s.Empty(results)
	s.Equal([]int{0}, missing)
}
