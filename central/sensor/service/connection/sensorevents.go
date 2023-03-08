package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const workerQueueSize = 16

var (
	deploymentEventType = reflectutils.Type((*central.SensorEvent_Deployment)(nil))
	nodeEventType       = reflectutils.Type((*central.SensorEvent_Node)(nil))
)

type sensorEventHandler struct {
	// workerQueues are keyed by central.SensorEvent type names.
	workerQueues      map[string]*workerQueue
	workerQueuesMutex sync.RWMutex

	deduper  *deduper
	pipeline pipeline.ClusterPipeline
	injector common.MessageInjector
	stopSig  *concurrency.ErrorSignal

	reconciliationMap *reconciliation.StoreMap
}

func newSensorEventHandler(pipeline pipeline.ClusterPipeline, injector common.MessageInjector, stopSig *concurrency.ErrorSignal, deduper *deduper) *sensorEventHandler {
	return &sensorEventHandler{
		workerQueues:      make(map[string]*workerQueue),
		reconciliationMap: reconciliation.NewStoreMap(),

		deduper:  deduper,
		pipeline: pipeline,
		injector: injector,
		stopSig:  stopSig,
	}
}

func (s *sensorEventHandler) handleMessages(ctx context.Context, msg *central.MsgFromSensor) error {
	return s.pipeline.Run(ctx, msg, s.injector)
}

func stripTypePrefix(s string) string {
	if idx := strings.LastIndex(s, "_"); idx != -1 {
		return s[idx+1:]
	}
	return s
}

func (s *sensorEventHandler) addMultiplexed(ctx context.Context, msg *central.MsgFromSensor) {
	var workerType string
	switch evt := msg.Msg.(type) {
	case *central.MsgFromSensor_Event:
		switch evt.Event.Resource.(type) {
		case *central.SensorEvent_Synced:
			// Call the reconcile functions
			if err := s.pipeline.Reconcile(ctx, s.reconciliationMap); err != nil {
				log.Errorf("error reconciling state: %v", err)
			}
			s.reconciliationMap.Close()
			return
		case *central.SensorEvent_ReprocessDeployment:
			workerType = deploymentEventType
		case *central.SensorEvent_NodeInventory:
			// This will put both NodeInventory and Node events in the same worker queue,
			// preventing events for the same Node ID to run concurrently.
			workerType = nodeEventType
			// Node and NodeInventory dedupe on Node ID. We use a different dedupe key for
			// NodeInventory because the two should not dedupe between themselves.
			msg.DedupeKey = fmt.Sprintf("NodeInventory:%s", msg.GetDedupeKey())
		default:
			// Default worker type is the event type.
			workerType = reflectutils.Type(evt.Event.Resource)
			if !s.reconciliationMap.IsClosed() {
				s.reconciliationMap.Add(evt.Event.Resource, evt.Event.Id)
			}
		}
	default:
		utils.Should(errors.New("handler only supports events"))
	}

	// If this is our first attempt at processing, then dedupe if we already processed this
	// If it is not our first attempt processing, then we need to check if a new version already
	// was processed
	if !s.deduper.shouldProcess(msg) {
		metrics.IncSensorEventsDeduper(true)
		return
	}
	metrics.IncSensorEventsDeduper(false)

	// Lazily create the queue for a type when not found.
	var queue *workerQueue
	concurrency.WithRLock(&s.workerQueuesMutex, func() {
		queue = s.workerQueues[workerType]
	})
	if queue == nil {
		concurrency.WithLock(&s.workerQueuesMutex, func() {
			queue = s.workerQueues[workerType]
			if queue == nil {
				queue = newWorkerQueue(workerQueueSize, stripTypePrefix(workerType), s.injector)
				s.workerQueues[workerType] = queue
				go queue.run(ctx, s.stopSig, s.handleMessages)
			}
		})
	}
	queue.push(msg)
}
