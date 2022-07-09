package postgres

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

func (s *clusterStoreImpl) Walk(_ context.Context, fn func(clusterID string, ts types.Timestamp, allFlows []*storage.NetworkFlow) error) error {
	return nil
}
