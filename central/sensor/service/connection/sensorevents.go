package connection

import (
	"context"
	"errors"
	"strings"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/reflectutils"
	"github.com/stackrox/rox/pkg/utils"
)

const workerQueueSize = 16

var deploymentQueueKey = reflectutils.Type((*central.SensorEvent_Deployment)(nil))

type sensorEventHandler struct {
	typeToQueue map[string]*workerQueue

	pipeline pipeline.ClusterPipeline
	injector common.MessageInjector
	stopSig  *concurrency.ErrorSignal

	reconciliationMap *reconciliation.StoreMap
}

func newSensorEventHandler(pipeline pipeline.ClusterPipeline, injector common.MessageInjector, stopSig *concurrency.ErrorSignal) *sensorEventHandler {
	return &sensorEventHandler{
		typeToQueue:       make(map[string]*workerQueue),
		reconciliationMap: reconciliation.NewStoreMap(),

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

func (s *sensorEventHandler) asyncCheckpointSender(id string) {
	WaitForCheckpoint(id)

	err := s.injector.InjectMessage(concurrency.Never(), &central.MsgToSensor{
		Msg: &central.MsgToSensor_ResourceCheckpoint{
			ResourceCheckpoint: &central.ResourceCheckpoint{
				Id: id,
			},
		},
	})
	if err != nil {
		log.Errorf("error sending checkpoint to sensor: %v", err)
	}
}

func (s *sensorEventHandler) addMultiplexed(ctx context.Context, msg *central.MsgFromSensor) {
	var typ string
	switch evt := msg.Msg.(type) {
	case *central.MsgFromSensor_Event:
		switch val := evt.Event.Resource.(type) {
		case *central.SensorEvent_Synced:
			// Call the reconcile functions
			if err := s.pipeline.Reconcile(ctx, s.reconciliationMap); err != nil {
				log.Errorf("error reconciling state: %v", err)
			}
			s.reconciliationMap.Close()
			return
		case *central.SensorEvent_Checkpoint:
			log.Infof("Received checkpoint: %+v", msg.GetEvent().GetCheckpoint().GetId())
			AddCheckpoint(msg.GetEvent().GetCheckpoint().GetId(), (workerQueueSize+1)*len(s.typeToQueue))
			for _, queue := range s.typeToQueue {
				queue.push(msg)
			}
			go s.asyncCheckpointSender(msg.GetEvent().GetCheckpoint().GetId())
			return
		case *central.SensorEvent_ReconciliationEvent_:
			if !s.reconciliationMap.IsClosed() {
				s.reconciliationMap.Add(val.ReconciliationEvent.GetType(), evt.Event.Id)
			}
			return
		case *central.SensorEvent_ReprocessDeployment:
			typ = deploymentQueueKey
		default:
			typ = reflectutils.Type(evt.Event.Resource)
			if !s.reconciliationMap.IsClosed() {
				s.reconciliationMap.Add(typ, evt.Event.Id)
			}
		}
	default:
		utils.Should(errors.New("handler only supports events"))
	}
	queue := s.typeToQueue[typ]
	// Lazily create the queue for a type if necessary
	if queue == nil {
		queue = newWorkerQueue(workerQueueSize, stripTypePrefix(typ))
		s.typeToQueue[typ] = queue
		go queue.run(ctx, s.stopSig, s.handleMessages)
	}
	queue.push(msg)
}
