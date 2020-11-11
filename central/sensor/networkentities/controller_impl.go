package networkentities

import (
	"context"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

type controller struct {
	clusterID      string
	netEntityMgr   common.NetworkEntityManager
	graphEvaluator graph.Evaluator
	requestSeqID   int64

	stopSig  concurrency.ReadOnlyErrorSignal
	injector common.MessageInjector

	lock sync.Mutex
}

func newController(clusterID string,
	netEntityMgr common.NetworkEntityManager,
	graphEvaluator graph.Evaluator,
	injector common.MessageInjector,
	stopSig concurrency.ReadOnlyErrorSignal) *controller {
	return &controller{
		clusterID:      clusterID,
		netEntityMgr:   netEntityMgr,
		graphEvaluator: graphEvaluator,
		stopSig:        stopSig,
		injector:       injector,
	}
}

func (c *controller) SyncNow(ctx context.Context) error {
	msg, err := c.getPushNetworkEntitiesRequestMsg(ctx)
	if err != nil {
		return err
	}

	// Stop early if the request message is outdated.
	if msg.GetPushNetworkEntitiesRequest().GetSeqID() != atomic.LoadInt64(&c.requestSeqID) {
		return nil
	}

	if err := c.injector.InjectMessage(ctx, msg); err != nil {
		return errors.Wrapf(err, "sending external network entities to cluster %q", c.clusterID)
	}

	// Increment network policy graph epoch indicating that an update to external sources could have changed the graph.
	c.graphEvaluator.IncrementEpoch(c.clusterID)
	return nil
}

func (c *controller) getPushNetworkEntitiesRequestMsg(ctx context.Context) (*central.MsgToSensor, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	requestSeqID := atomic.AddInt64(&c.requestSeqID, 1)

	netEntities, err := c.netEntityMgr.GetAllEntitiesForCluster(ctx, c.clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "obtaining external network entities to sync with cluster %q", c.clusterID)
	}

	srcs := make([]*storage.NetworkEntityInfo, 0, len(netEntities))
	for _, entity := range netEntities {
		srcs = append(srcs, entity.GetInfo())
	}

	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_PushNetworkEntitiesRequest{
			PushNetworkEntitiesRequest: &central.PushNetworkEntitiesRequest{
				Entities: srcs,
				SeqID:    requestSeqID,
			},
		},
	}, nil
}
