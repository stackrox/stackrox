package processsignal

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/channelmultiplexer"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

var (
	log = logging.LoggerForModule()
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
	pubSubDispatcher   common.PubSubDispatcher
	stopper            concurrency.Stopper
	// enricher context
	cancelEnricherCtx context.CancelCauseFunc
}

// NewProcessPipeline defines how to process a ProcessIndicator
func NewProcessPipeline(indicators chan *message.ExpiringMessage, clusterEntities *clusterentities.Store, processFilter filter.Filter, detector detector.Detector, pubSubDispatcher common.PubSubDispatcher) *Pipeline {
	log.Debug("Calling NewProcessPipeline")
	enricherCtx, cancelEnricherCtx := context.WithCancelCause(context.Background())
	en := newEnricher(enricherCtx, clusterEntities, pubSubDispatcher)

	p := &Pipeline{
		clusterEntities:    clusterEntities,
		indicators:         indicators,
		enricher:           en,
		processFilter:      processFilter,
		detector:           detector,
		pubSubDispatcher:   pubSubDispatcher,
		cancelEnricherCtx:  cancelEnricherCtx,
		stopper:            concurrency.NewStopper(),
	}

	// Dual-mode initialization based on feature flag
	if features.SensorInternalPubSub.Enabled() && pubSubDispatcher != nil {
		log.Info("Process pipeline using pub/sub mode")
		if err := pubSubDispatcher.RegisterConsumerToLane(pubsub.EnrichedProcessIndicatorTopic, pubsub.EnrichedProcessIndicatorLane, p.processEnrichedIndicator); err != nil {
			log.Errorf("Failed to register consumer for enriched process indicators: %v", err)
		}
	} else {
		log.Info("Process pipeline using legacy channel mode")
		enrichedIndicators := make(chan *storage.ProcessIndicator)
		p.enrichedIndicators = enrichedIndicators

		cm := channelmultiplexer.NewMultiplexer[*storage.ProcessIndicator]()
		cm.AddChannel(en.getEnrichedC())  // PIs that are enriched in the enricher
		cm.AddChannel(enrichedIndicators) // PIs that are enriched directly in the pipeline
		cm.Run()
		p.cm = cm

		go p.sendIndicatorEvent()
	}

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
		// Only close enrichedIndicators channel in legacy mode
		if p.enrichedIndicators != nil {
			close(p.enrichedIndicators)
		}
		_ = p.enricher.Stopped().Wait()
		_ = p.stopper.Client().Stopped().Wait()
	}()
	p.stopper.Client().Stop()
}

// WaitForShutdown waits for the pipeline shutdown to complete.
// This is useful for tests that need to ensure shutdown has fully completed.
func (p *Pipeline) WaitForShutdown() error {
	if err := p.enricher.Stopped().Wait(); err != nil {
		return errors.Wrap(err, "waiting for enricher to stop")
	}
	if err := p.stopper.Client().Stopped().Wait(); err != nil {
		return errors.Wrap(err, "waiting for pipeline stopper")
	}
	return nil
}

// Notify allows the component state to be propagated to the pipeline
func (p *Pipeline) Notify(e common.SensorComponentEvent) {
	// With event buffering enabled, we use long-lived contexts and don't cancel on disconnect
	log.Info(common.LogSensorComponentEvent(e))
}

// Process defines processes to process a ProcessIndicator
// If the pipeline is shutting down, the signal is dropped to prevent sending on closed channels.
func (p *Pipeline) Process(signal *storage.ProcessSignal) {
	select {
	case <-p.stopper.Flow().StopRequested():
		p.dropIndicator(signal, "pipeline shutting down before enrichment")
		return
	default:
	}

	indicator := &storage.ProcessIndicator{
		Id:     uuid.NewV4().String(),
		Signal: signal,
	}

	if features.SensorInternalPubSub.Enabled() && p.pubSubDispatcher != nil {
		event := NewUnenrichedProcessIndicatorEvent(context.Background(), indicator)
		if err := p.pubSubDispatcher.Publish(event); err != nil {
			metrics.IncrementProcessSignalDroppedCount()
			log.Errorf("Failed to publish unenriched process indicator for container %s with id %s: %v",
				signal.GetContainerId(), indicator.GetId(), err)
		}
	} else {
		p.enricher.add(indicator)
	}
}

func (p *Pipeline) dropIndicator(signal *storage.ProcessSignal, reason string) {
	metrics.IncrementProcessSignalDroppedCount()
	if signal != nil {
		log.Debugf("Dropping process signal for container %s: %s", signal.GetContainerId(), reason)
	} else {
		log.Debugf("Dropping process signal: %s", reason)
	}
}

func (p *Pipeline) sendIndicatorEvent() {
	defer p.stopper.Flow().ReportStopped()
	for indicator := range p.cm.GetOutput() {
		if !p.processFilter.Add(indicator) {
			continue
		}
		p.detector.ProcessIndicator(context.Background(), indicator)
		p.sendToCentral(
			message.NewExpiring(context.Background(), &central.MsgFromSensor{
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

// publishEnrichedIndicator is used in pub/sub mode instead of sending to the enrichedIndicators channel.
func (p *Pipeline) publishEnrichedIndicator(ctx context.Context, indicator *storage.ProcessIndicator) {
	if p.pubSubDispatcher == nil {
		log.Error("Cannot publish enriched indicator: pub/sub dispatcher is nil")
		return
	}

	event := NewEnrichedProcessIndicatorEvent(ctx, indicator)
	if err := p.pubSubDispatcher.Publish(event); err != nil {
		metrics.IncrementProcessSignalDroppedCount()
		log.Errorf("Failed to publish enriched process indicator for deployment %s with id %s: %v",
			indicator.GetDeploymentId(), indicator.GetId(), err)
	}
}

// processEnrichedIndicator replaces the sendIndicatorEvent goroutine in legacy mode.
func (p *Pipeline) processEnrichedIndicator(event pubsub.Event) error {
	enrichedEvent, ok := event.(*EnrichedProcessIndicatorEvent)
	if !ok {
		log.Errorf("Received unexpected event type: %T", event)
		return errors.Errorf("unexpected event type: %T", event)
	}

	indicator := enrichedEvent.Indicator
	if indicator == nil {
		return errors.New("enriched process indicator event has nil indicator")
	}

	if !p.processFilter.Add(indicator) {
		return nil
	}

	p.detector.ProcessIndicator(enrichedEvent.Context, indicator)

	p.sendToCentral(
		message.NewExpiring(enrichedEvent.Context, &central.MsgFromSensor{
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
	return nil
}
