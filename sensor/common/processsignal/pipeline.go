package processsignal

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/channelmultiplexer"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/process/normalize"
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
	ctxMux             *sync.Mutex
	ctx                context.Context
	ctxCancel          context.CancelCauseFunc
}

// NewProcessPipeline defines how to process a ProcessIndicator
func NewProcessPipeline(indicators chan *message.ExpiringMessage, clusterEntities *clusterentities.Store, processFilter filter.Filter, detector detector.Detector) *Pipeline {
	log.Debug("Calling NewProcessPipeline")
	ctx, cancel := context.WithCancelCause(context.Background())
	en := newEnricher(ctx, clusterEntities)
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
		ctxMux:             &sync.Mutex{},
		ctx:                ctx,
		ctxCancel:          cancel,
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
	indicator.ContainerStartTime = cachedContainer.StartTime
	indicator.ImageId = cachedContainer.ImageID
}

// Shutdown closes all communication channels and shutdowns the enricher
func (p *Pipeline) Shutdown() {
	p.cancelCurrentContext()
	defer close(p.enrichedIndicators)
}

// Notify allows the component state to be propagated to the pipeline
func (p *Pipeline) Notify(e common.SensorComponentEvent) {
	log.Debugf("Received notify: %s", e)
	switch e {
	case common.SensorComponentEventCentralReachable:
		p.createNewContext()
		log.Debug("ProcessSingnalPipeline runs now in Online mode.")
	case common.SensorComponentEventOfflineMode:
		p.cancelCurrentContext()
		log.Debug("ProcessSingnalPipeline runs now in Offline mode.")
	}
}

func (p *Pipeline) createNewContext() {
	p.ctxMux.Lock()
	defer p.ctxMux.Unlock()
	p.ctx, p.ctxCancel = context.WithCancelCause(context.Background())
}

func (p *Pipeline) getCurrentContext() context.Context {
	p.ctxMux.Lock()
	defer p.ctxMux.Unlock()
	return p.ctx
}

func (p *Pipeline) cancelCurrentContext() {
	p.ctxMux.Lock()
	defer p.ctxMux.Unlock()
	if p.ctxCancel != nil {
		p.ctxCancel(errSensorOffline)
	}
}

// Process defines processes to process a ProcessIndicator
func (p *Pipeline) Process(signal *storage.ProcessSignal) {
	indicator := &storage.ProcessIndicator{
		Id:     uuid.NewV4().String(),
		Signal: signal,
	}

	// indicator.GetSignal() is never nil at this point
	metadata, ok := p.clusterEntities.LookupByContainerID(indicator.GetSignal().GetContainerId())
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
	for indicator := range p.cm.GetOutput() {
		if !p.processFilter.Add(indicator) {
			continue
		}
		p.detector.ProcessIndicator(indicator)
		p.indicators <- message.NewExpiring(p.getCurrentContext(), &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Id:     indicator.GetId(),
					Action: central.ResourceAction_CREATE_RESOURCE,
					Resource: &central.SensorEvent_ProcessIndicator{
						ProcessIndicator: indicator,
					},
				},
			},
		})
	}
}
