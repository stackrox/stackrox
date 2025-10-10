package processsignal

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/channelmultiplexer"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/process/normalize"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
)

var (
	log              = logging.LoggerForModule()
	errSensorOffline = errors.New("sensor is offline")
)

// Pipeline is the struct that handles a process signal
type Pipeline struct {
	clusterEntities    *clusterentities.Store
	indicators         chan *message.ExpiringMessage
	enrichedIndicators chan *storage.ProcessIndicator
	enricher           *enricher
	processFilter      filter.Filter
	detector           detector.Detector
	cm                 *channelmultiplexer.ChannelMultiplexer[*storage.ProcessIndicator]
	stopper            concurrency.Stopper
	// enricher context
	cancelEnricherCtx context.CancelCauseFunc
	// message context
	msgCtxMux    *sync.Mutex
	msgCtx       context.Context
	msgCtxCancel context.CancelCauseFunc
}

// NewProcessPipeline defines how to process a ProcessIndicator
func NewProcessPipeline(indicators chan *message.ExpiringMessage, clusterEntities *clusterentities.Store, processFilter filter.Filter, detector detector.Detector) *Pipeline {
	log.Debug("Calling NewProcessPipeline")
	msgCtx, cancelMsgCtx := context.WithCancelCause(context.Background())
	enricherCtx, cancelEnricherCtx := context.WithCancelCause(context.Background())
	en := newEnricher(enricherCtx, clusterEntities)
	enrichedIndicators := make(chan *storage.ProcessIndicator)

	cm := channelmultiplexer.NewMultiplexer[*storage.ProcessIndicator]()
	cm.AddChannel(en.getEnrichedC())  // PIs that are enriched in the enricher
	cm.AddChannel(enrichedIndicators) // PIs that are enriched directly in the pipeline
	cm.Run()
	p := &Pipeline{
		clusterEntities:    clusterEntities,
		indicators:         indicators,
		enricher:           en,
		enrichedIndicators: enrichedIndicators,
		processFilter:      processFilter,
		detector:           detector,
		cm:                 cm,
		cancelEnricherCtx:  cancelEnricherCtx,
		msgCtxMux:          &sync.Mutex{},
		msgCtx:             msgCtx,
		msgCtxCancel:       cancelMsgCtx,
		stopper:            concurrency.NewStopper(),
	}
	go p.sendIndicatorEvent()
	return p
}

func populateIndicatorFromCachedContainer(indicator *storage.ProcessIndicator, cachedContainer clusterentities.ContainerMetadata) {
	indicator.DeploymentId = cachedContainer.DeploymentID
	indicator.ContainerName = cachedContainer.ContainerName
	indicator.PodId = cachedContainer.PodID
	indicator.PodUid = cachedContainer.PodUID
	indicator.Namespace = cachedContainer.Namespace
	indicator.ContainerStartTime = protocompat.ConvertTimeToTimestampOrNil(cachedContainer.StartTime)
	indicator.ImageId = cachedContainer.ImageID
}

// Shutdown closes all communication channels and shutdowns the enricher
func (p *Pipeline) Shutdown() {
	p.cancelEnricherCtx(errors.New("pipeline shutdown"))
	defer func() {
		close(p.enrichedIndicators)
		_ = p.enricher.Stopped().Wait()
		_ = p.stopper.Client().Stopped().Wait()
	}()
	p.stopper.Client().Stop()
}

// Notify allows the component state to be propagated to the pipeline
func (p *Pipeline) Notify(e common.SensorComponentEvent) {
	// With event buffering enabled, we use long-lived contexts and don't cancel on disconnect
	log.Info(common.LogSensorComponentEvent(e))
}

func (p *Pipeline) getCurrentContext() context.Context {
	// Use long-lived context that persists across disconnects for event buffering
	return context.Background()
}

// Process defines processes to process a ProcessIndicator
func (p *Pipeline) Process(signal *storage.ProcessSignal) {
	indicator := &storage.ProcessIndicator{
		Id:     uuid.NewV4().String(),
		Signal: signal,
	}

	// indicator.GetSignal() is never nil at this point
	metadata, ok, _ := p.clusterEntities.LookupByContainerID(indicator.GetSignal().GetContainerId())
	if !ok {
		p.enricher.Add(indicator)
		return
	}
	metrics.IncrementProcessEnrichmentHits()
	populateIndicatorFromCachedContainer(indicator, metadata)
	normalize.Indicator(indicator)
	p.enrichedIndicators <- indicator
}

func (p *Pipeline) sendIndicatorEvent() {
	defer p.stopper.Flow().ReportStopped()
	for indicator := range p.cm.GetOutput() {
		if !p.processFilter.Add(indicator) {
			continue
		}
		p.detector.ProcessIndicator(p.getCurrentContext(), indicator)
		p.sendToCentral(
			message.NewExpiring(p.getCurrentContext(), &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Id:     indicator.GetId(),
						Action: central.ResourceAction_CREATE_RESOURCE,
						Resource: &central.SensorEvent_ProcessIndicator{
							ProcessIndicator: indicator,
						},
					},
				},
			}),
		)
		metrics.SetProcessSignalBufferSizeGauge(len(p.indicators))
	}
}

func (p *Pipeline) sendToCentral(msg *message.ExpiringMessage) {
	select {
	case p.indicators <- msg:
	case <-p.stopper.Flow().StopRequested():
		return
	default:
		metrics.IncrementProcessSignalDroppedCount()
		log.Errorf("The output channel is full. Dropping process indicator event for deployment %s with id %s and process name %s",
			msg.GetEvent().GetProcessIndicator().GetDeploymentId(),
			msg.GetEvent().GetProcessIndicator().GetId(),
			msg.GetEvent().GetProcessIndicator().GetSignal().GetName())
	}
}
