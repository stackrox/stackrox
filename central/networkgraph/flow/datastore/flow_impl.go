package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkgraph/aggregator"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
)

var (
	log             = logging.LoggerForModule()
	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
)

type flowDataStoreImpl struct {
	storage                   store.FlowStore
	graphConfig               graphConfigDS.DataStore
	hideDefaultExtSrcsManager aggregator.NetworkConnsAggregator
	deletedDeploymentsCache   expiringcache.Cache
}

func (fds *flowDataStoreImpl) GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
	flows, ts, err := fds.storage.GetAllFlows(ctx, since)
	if err != nil {
		return nil, types.Timestamp{}, nil
	}

	flows, err = fds.adjustFlowsForGraphConfig(ctx, flows)
	if err != nil {
		return nil, types.Timestamp{}, err
	}
	return flows, ts, nil
}

func (fds *flowDataStoreImpl) GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
	flows, ts, err := fds.storage.GetMatchingFlows(ctx, pred, since)
	if err != nil {
		return nil, types.Timestamp{}, nil
	}

	flows, err = fds.adjustFlowsForGraphConfig(ctx, flows)
	if err != nil {
		return nil, types.Timestamp{}, err
	}
	return flows, ts, nil
}

func (fds *flowDataStoreImpl) adjustFlowsForGraphConfig(ctx context.Context, flows []*storage.NetworkFlow) ([]*storage.NetworkFlow, error) {
	graphConfigReadCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraphConfig)))

	config, err := fds.graphConfig.GetNetworkGraphConfig(graphConfigReadCtx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain network graph configuration")
	}

	if config.GetHideDefaultExternalSrcs() {
		return fds.hideDefaultExtSrcsManager.Aggregate(flows), nil
	}
	return flows, nil
}

func (fds *flowDataStoreImpl) isDeletedDeployment(id string) bool {
	deleted, _ := fds.deletedDeploymentsCache.Get(id).(bool)
	return deleted
}

func (fds *flowDataStoreImpl) UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error {
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
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

	return fds.storage.UpsertFlows(ctx, filtered, lastUpdateTS)
}

func (fds *flowDataStoreImpl) RemoveFlowsForDeployment(ctx context.Context, id string) error {
	// This is reached only on write access to deployment,
	// therefore no need to fetch the deployment again for access check
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return fds.storage.RemoveFlowsForDeployment(ctx, id)
}

func (fds *flowDataStoreImpl) RemoveMatchingFlows(ctx context.Context, keyMatchFn func(props *storage.NetworkFlowProperties) bool, valueMatchFn func(flow *storage.NetworkFlow) bool) error {
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return fds.storage.RemoveMatchingFlows(ctx, keyMatchFn, valueMatchFn)
}
