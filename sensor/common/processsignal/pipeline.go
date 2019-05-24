package processsignal

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/metrics"
)

var (
	log = logging.LoggerForModule()
)

// Pipeline is the struct that handles a process signal
type Pipeline struct {
	clusterEntities    *clusterentities.Store
	indicators         chan *central.SensorEvent
	enrichedIndicators chan *storage.ProcessIndicator
	deduper            *deduper
	enricher           *enricher
}

// NewProcessPipeline defines how to process a ProcessIndicator
func NewProcessPipeline(indicators chan *central.SensorEvent, clusterEntities *clusterentities.Store) *Pipeline {
	enrichedIndicators := make(chan *storage.ProcessIndicator)
	p := &Pipeline{
		clusterEntities:    clusterEntities,
		indicators:         indicators,
		deduper:            newDeduper(),
		enricher:           newEnricher(clusterEntities, enrichedIndicators),
		enrichedIndicators: enrichedIndicators,
	}
	go p.sendIndicatorEvent()
	return p
}

func populateIndicatorFromCachedContainer(indicator *storage.ProcessIndicator, cachedContainer clusterentities.ContainerMetadata) {
	indicator.DeploymentId = cachedContainer.DeploymentID
	indicator.ContainerName = cachedContainer.ContainerName
	indicator.PodId = cachedContainer.PodID
	indicator.Namespace = cachedContainer.Namespace
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
	p.enrichedIndicators <- indicator
}

func (p *Pipeline) sendIndicatorEvent() {
	for indicator := range p.enrichedIndicators {
		// determine whether or not we should send the event
		if !p.deduper.Allow(indicator) {
			continue
		}
		p.indicators <- &central.SensorEvent{
			Id:     indicator.GetId(),
			Action: central.ResourceAction_CREATE_RESOURCE,
			Resource: &central.SensorEvent_ProcessIndicator{
				ProcessIndicator: indicator,
			},
		}
	}
}
