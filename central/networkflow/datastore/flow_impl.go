package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
)

var (
	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
)

type flowDataStoreImpl struct {
	storage                 store.FlowStore
	deletedDeploymentsCache expiringcache.Cache
}

func (fds *flowDataStoreImpl) GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
	return fds.storage.GetAllFlows(since)
}

func (fds *flowDataStoreImpl) GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
	return fds.storage.GetMatchingFlows(pred, since)
}

func (fds *flowDataStoreImpl) isDeletedDeployment(id string) bool {
	deleted, _ := fds.deletedDeploymentsCache.Get(id).(bool)
	return deleted
}

func (fds *flowDataStoreImpl) UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error {
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	filtered := flows[:0]
	for _, flow := range flows {
		if flow.GetProps().GetSrcEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && fds.isDeletedDeployment(flow.GetProps().GetSrcEntity().GetId()) {
			continue
		}
		if flow.GetProps().GetDstEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && fds.isDeletedDeployment(flow.GetProps().GetDstEntity().GetId()) {
			continue
		}
		filtered = append(filtered, flow)
	}

	return fds.storage.UpsertFlows(filtered, lastUpdateTS)
}

func (fds *flowDataStoreImpl) RemoveFlowsForDeployment(ctx context.Context, id string) error {
	// This is reached only on write access to deployment,
	// therefore no need to fetch the deployment again for access check
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return fds.storage.RemoveFlowsForDeployment(id)
}
