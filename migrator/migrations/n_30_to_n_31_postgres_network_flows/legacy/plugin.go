package legacy

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/common"
	"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/store"
)

func (s *clusterStoreImpl) Walk(ctx context.Context, fn func(clusterID string, ts types.Timestamp, allFlows []*storage.NetworkFlow) error) error {
	iterator := s.db.NewIterator(readOptions)
	defer iterator.Close()
	// Runs are sorted by time so we must iterate over each key to see if it has the correct run ID.
	prefix := []byte(common.GlobalPrefix)
	var currentCluster string
	var currentFlowStore store.FlowStore
	for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
		clusterID, err := common.GetClusterIDFromKey(iterator.Key().Data())
		if err != nil {
			log.WriteToStderrf("%v", err)
			continue
		}
		if string(clusterID) == currentCluster {
			continue
		}
		currentCluster = string(clusterID)
		currentFlowStore, err = s.CreateFlowStore(ctx, currentCluster)
		flows, ts, err := currentFlowStore.GetAllFlows(ctx, &types.Timestamp{})
		if err = fn(currentCluster, ts, flows); err != nil {
			return err
		}
	}
	return nil
}
