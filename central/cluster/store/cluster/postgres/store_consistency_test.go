//go:build sql_integration

package postgres

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
)

func (s *ClustersStoreSuite) TestReadsObserveOtherStoreWrites() {
	ctx := sac.WithAllAccess(context.Background())
	writer := New(s.testDB.DB)
	cluster := &storage.Cluster{Id: uuid.NewV4().String(), Name: "shared-cluster", Labels: map[string]string{"team": "initial"}}
	_, found, err := s.store.Get(ctx, cluster.GetId())
	s.Require().NoError(err)
	s.Require().False(found)

	s.Require().NoError(writer.Upsert(ctx, cluster))
	result, found, err := s.store.Get(ctx, cluster.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Equal(cluster.GetLabels(), result.GetLabels())

	cluster.Labels = map[string]string{"team": "updated"}
	s.Require().NoError(writer.Upsert(ctx, cluster))
	results, missing, err := s.store.GetMany(ctx, []string{cluster.GetId()})
	s.Require().NoError(err)
	s.Require().Empty(missing)
	s.Require().Len(results, 1)
	s.Equal(cluster.GetLabels(), results[0].GetLabels())

	s.Require().NoError(writer.Delete(ctx, cluster.GetId()))
	_, found, err = s.store.Get(ctx, cluster.GetId())
	s.Require().NoError(err)
	s.False(found)
	results, missing, err = s.store.GetMany(ctx, []string{cluster.GetId()})
	s.Require().NoError(err)
	s.Empty(results)
	s.Equal([]int{0}, missing)
}
