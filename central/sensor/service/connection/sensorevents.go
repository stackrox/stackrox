package connection

import (
	"context"
	"fmt"
	"strings"

	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/deduperkey"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
)

const workerQueueSize = 16

var (
	deploymentEventType = reflectutils.Type((*central.SensorEvent_Deployment)(nil))
	nodeEventType       = reflectutils.Type((*central.SensorEvent_Node)(nil))
)

type sensorEventHandler struct {
	cluster       *storage.Cluster
	sensorVersion string
	// workerQueues are keyed by central.SensorEvent type names.
	workerQueues      map[string]*workerQueue
	workerQueuesMutex sync.RWMutex

	deduper     hashManager.Deduper
	initSyncMgr *initSyncManager
	pipeline    pipeline.ClusterPipeline
	injector    common.MessageInjector
	stopSig     *concurrency.ErrorSignal

	reconciliationMap *reconciliation.StoreMap
}

func newSensorEventHandler(
	cluster *storage.Cluster,
	sensorVersion string,
	pipeline pipeline.ClusterPipeline,
	injector common.MessageInjector,
	stopSig *concurrency.ErrorSignal,
	deduper hashManager.Deduper,
	initSyncMgr *initSyncManager,
) *sensorEventHandler {
	return &sensorEventHandler{
		cluster:       cluster,
		sensorVersion: sensorVersion,

		workerQueues:      make(map[string]*workerQueue),
		reconciliationMap: reconciliation.NewStoreMap(),

		deduper:     deduper,
		initSyncMgr: initSyncMgr,
		pipeline:    pipeline,
		injector:    injector,
		stopSig:     stopSig,
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
	event := msg.GetEvent()
	if event == nil {
		log.Errorf("Received unknown msg from cluster %s (%s) of type %T. May be due to Sensor (%s) version mismatch with Central (%s)", s.cluster.GetName(), s.cluster.GetId(), msg.Msg, s.sensorVersion, version.GetMainVersion())
		return
	}

	var workerType string
	switch event.Resource.(type) {
	// The occurrence of a "Synced" event from the sensor marks the conclusion
	// of the initial synchronization process.
	case *central.SensorEvent_Synced:
		// Call the reconcile functions
		log.Info("Receiving reconciliation event")

		unchangedIDs := event.GetSynced().GetUnchangedIds()
		if unchangedIDs != nil {
			parsedKeys, err := deduperkey.ParseKeySlice(unchangedIDs)
			if err != nil {
				// Show warning for failed keys
				log.Warnf("Error parsing %d unchanged IDs: %s", len(unchangedIDs), err)
			}
			for _, k := range parsedKeys {
				s.reconciliationMap.AddWithTypeString(k.ResourceType.String(), k.ID)
			}
		}

		if err := s.pipeline.Reconcile(ctx, s.reconciliationMap); err != nil {
			log.Errorf("error reconciling state: %v", err)
		}
		s.initSyncMgr.Remove(s.cluster.GetId())
		s.deduper.ProcessSync()
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
	case *central.SensorEvent_ComplianceOperatorResultV2:
		if !features.ComplianceEnhancements.Enabled() {
			log.Warnf("Received next gen compliance event from cluster %s (%s). Next gen compliance is disabled on central.", s.cluster.GetName(), s.cluster.GetId())
			return
		}
		// Due to needing both V1 and V2 compliance to run at the same time and due to how the
		// reconciliation keys are used we need to use the V1 key for reconciliation.  This could
		// have been avoided by sending both messages from sensor during the transition, but
		// that seemed like a lot of extra traffic.
		workerType = reflectutils.Type(&central.SensorEvent_ComplianceOperatorResult{})
		if !s.reconciliationMap.IsClosed() {
			s.reconciliationMap.Add(&central.SensorEvent_ComplianceOperatorResult{}, event.Id)
		}
	case *central.SensorEvent_ComplianceOperatorProfileV2:
		if !features.ComplianceEnhancements.Enabled() {
			log.Warnf("Received next gen compliance event from cluster %s (%s). Next gen compliance is disabled on central.", s.cluster.GetName(), s.cluster.GetId())
			return
		}
		// Due to needing both V1 and V2 compliance to run at the same time and due to how the
		// reconciliation keys are used we need to use the V1 key for reconciliation.  This could
		// have been avoided by sending both messages from sensor during the transition, but
		// that seemed like a lot of extra traffic.
		workerType = reflectutils.Type(&central.SensorEvent_ComplianceOperatorProfile{})
		if !s.reconciliationMap.IsClosed() {
			s.reconciliationMap.Add(&central.SensorEvent_ComplianceOperatorProfile{}, event.Id)
		}
	default:
		if event.GetResource() == nil {
			log.Errorf("Received event with unknown resource from cluster %s (%s). May be due to Sensor (%s) version mismatch with Central (%s)", s.cluster.GetName(), s.cluster.GetId(), s.sensorVersion, version.GetMainVersion())
			return
		}

		// Default worker type is the event type.
		workerType = reflectutils.Type(event.Resource)
		if !s.reconciliationMap.IsClosed() {
			s.reconciliationMap.Add(event.Resource, event.Id)
		}
	}

	// If this is our first attempt at processing, then dedupe if we already processed this
	// If it is not our first attempt processing, then we need to check if a new version already
	// was processed
	if !s.deduper.ShouldProcess(msg) {
		metrics.IncSensorEventsDeduper(true, msg)
		return
	}
	metrics.IncSensorEventsDeduper(false, msg)

	// Lazily create the queue for a type when not found.
	queue := concurrency.WithRLock1(&s.workerQueuesMutex, func() *workerQueue {
		return s.workerQueues[workerType]
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
