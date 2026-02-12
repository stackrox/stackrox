package processsignal

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/channelmultiplexer"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
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

type processPipeline interface {
	Start() error
	Stop() error
	Process(signal *storage.ProcessIndicator)
}

type basePipeline struct {
	processFilter     filter.Filter
	detector          detector.Detector
	stopper           concurrency.Stopper
	cancelEnricherCtx context.CancelCauseFunc
	enricher          *enricher
	indicators        chan *message.ExpiringMessage
}

func newBasePipeline(indicators chan *message.ExpiringMessage, enricher *enricher, processFilter filter.Filter, detector detector.Detector, stopper concurrency.Stopper, cancelEnricherCtx context.CancelCauseFunc) basePipeline {
	return basePipeline{
		processFilter:     processFilter,
		detector:          detector,
		stopper:           stopper,
		enricher:          enricher,
		cancelEnricherCtx: cancelEnricherCtx,
		indicators:        indicators,
	}
}

func (b *basePipeline) Start() error {
	return nil
}

func (b *basePipeline) Stop() error {
	b.cancelEnricherCtx(errors.New("pipeline shutdown"))
	b.stopper.Flow().ReportStopped()
	return nil
}

func (p *basePipeline) sendToCentral(msg *message.ExpiringMessage) {
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

// processEnrichedIndicator replaces the sendIndicatorEvent goroutine in legacy mode.
func (p *basePipeline) processEnrichedIndicator(event pubsub.Event) error {
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

	p.handleEnrichedIndicator(enrichedEvent.Context, indicator)
	return nil
}

func (p *basePipeline) handleEnrichedIndicator(ctx context.Context, indicator *storage.ProcessIndicator) {
	p.detector.ProcessIndicator(ctx, indicator)

	p.sendToCentral(
		message.NewExpiring(ctx, &central.MsgFromSensor{
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

type channelPipeline struct {
	basePipeline

	cm                 *channelmultiplexer.ChannelMultiplexer[*storage.ProcessIndicator]
	clusterEntities    *clusterentities.Store
	enrichedIndicators chan *storage.ProcessIndicator
}

func newChannelPipeline(base basePipeline, clusterEntities *clusterentities.Store) processPipeline {
	enrichedIndicators := make(chan *storage.ProcessIndicator)

	cm := channelmultiplexer.NewMultiplexer[*storage.ProcessIndicator]()
	cm.AddChannel(base.enricher.getEnrichedC()) // PIs that are enriched in the enricher
	cm.AddChannel(enrichedIndicators)           // PIs that are enriched directly in the pipeline
	cm.Run()

	return &channelPipeline{
		basePipeline:       base,
		cm:                 cm,
		clusterEntities:    clusterEntities,
		enrichedIndicators: enrichedIndicators,
	}
}

func (p *channelPipeline) Start() error {
	errlist := errorhelpers.NewErrorList("starting channel pipeline")
	if err := p.basePipeline.Start(); err != nil {
		errlist.AddError(err)
	}

	go p.sendIndicatorEvent()
	return errlist.ToError()
}

func (c *channelPipeline) sendIndicatorEvent() {
	defer c.stopper.Flow().ReportStopped()
	for indicator := range c.cm.GetOutput() {
		if err := c.processEnrichedIndicator(NewEnrichedProcessIndicatorEvent(context.Background(), indicator)); err != nil {
			log.Errorf("failed to process enriched indicator: %v", err)
		}
	}
}

func (p *channelPipeline) Stop() error {
	errlist := errorhelpers.NewErrorList("stopping channel pipeline")
	if err := p.basePipeline.Stop(); err != nil {
		errlist.AddError(err)
	}

	return errlist.ToError()
}

func (p *channelPipeline) Process(indicator *storage.ProcessIndicator) {
	p.enricher.add(indicator)
}

type pubsubPipeline struct {
	basePipeline
	pubSubDispatcher common.PubSubDispatcher
}

func newPubSubPipeline(base basePipeline, pubSubDispatcher common.PubSubDispatcher) processPipeline {
	return &pubsubPipeline{
		basePipeline:     base,
		pubSubDispatcher: pubSubDispatcher,
	}
}

func (p *pubsubPipeline) Start() error {
	errlist := errorhelpers.NewErrorList("starting pubsub pipeline")
	if err := p.basePipeline.Start(); err != nil {
		errlist.AddError(err)
	}

	if err := p.pubSubDispatcher.RegisterConsumerToLane(
		pubsub.EnrichedProcessConsumer,
		pubsub.EnrichedProcessIndicatorTopic,
		pubsub.EnrichedProcessIndicatorLane,
		p.processEnrichedIndicator,
	); err != nil {
		errlist.AddWrap(err, "Failed to register consumer for enriched process indicators")
	}

	return errlist.ToError()
}

func (p *pubsubPipeline) Stop() error {
	errlist := errorhelpers.NewErrorList("stopping pubsub pipeline")
	if err := p.basePipeline.Stop(); err != nil {
		errlist.AddError(err)
	}

	return errlist.ToError()
}

func (p *pubsubPipeline) Process(indicator *storage.ProcessIndicator) {
	event := NewUnenrichedProcessIndicatorEvent(context.Background(), indicator)
	if err := p.pubSubDispatcher.Publish(event); err != nil {
		dropSignal(indicator.GetSignal(), "Failed to published to dispatcher")
	}
}

// Pipeline is the struct that handles a process signal
type Pipeline struct {
	inner   processPipeline
	stopper concurrency.Stopper
}

// NewProcessPipeline defines how to process a ProcessIndicator
func NewProcessPipeline(indicators chan *message.ExpiringMessage, clusterEntities *clusterentities.Store, processFilter filter.Filter, detector detector.Detector, pubSubDispatcher common.PubSubDispatcher) *Pipeline {
	log.Debug("Calling NewProcessPipeline")

	enricherCtx, cancelEnricherCtx := context.WithCancelCause(context.Background())
	en := newEnricher(enricherCtx, clusterEntities, pubSubDispatcher)

	stopper := concurrency.NewStopper()
	base := newBasePipeline(indicators, en, processFilter, detector, stopper, cancelEnricherCtx)

	var inner processPipeline
	if features.SensorInternalPubSub.Enabled() && pubSubDispatcher != nil {
		log.Info("Process pipeline using pub/sub mode")
		inner = newPubSubPipeline(base, pubSubDispatcher)
	} else {
		log.Info("Process pipeline using legacy channel mode")
		inner = newChannelPipeline(base, clusterEntities)
	}

	return &Pipeline{
		inner:   inner,
		stopper: stopper,
	}
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
	defer func() {
		_ = p.inner.Stop()
		_ = p.stopper.Client().Stopped().Wait()
	}()

	p.stopper.Client().Stop()
}

// WaitForShutdown waits for the pipeline shutdown to complete.
// This is useful for tests that need to ensure shutdown has fully completed.
func (p *Pipeline) WaitForShutdown() error {
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
		dropSignal(signal, "pipeline shutting down before enrichment")
		return
	default:
	}

	indicator := &storage.ProcessIndicator{
		Id:     uuid.NewV4().String(),
		Signal: signal,
	}
	p.inner.Process(indicator)
}

func dropSignal(signal *storage.ProcessSignal, reason string) {
	metrics.IncrementProcessSignalDroppedCount()
	if signal != nil {
		log.Debugf("Dropping process signal for container %s: %s", signal.GetContainerId(), reason)
	} else {
		log.Debugf("Dropping process signal: %s", reason)
	}
}
