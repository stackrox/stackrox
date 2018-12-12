package processsignal

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/metrics"
)

var logger = logging.LoggerForModule()

// Pipeline is the struct that handles a process signal
type Pipeline struct {
	clusterEntities    *clusterentities.Store
	indicators         chan *v1.SensorEvent
	enrichedIndicators chan *v1.ProcessIndicator
	deduper            *deduper
	enricher           *enricher
}

// NewProcessPipeline defines how to process a ProcessIndicator
func NewProcessPipeline(indicators chan *v1.SensorEvent, clusterEntities *clusterentities.Store) *Pipeline {
	enrichedIndicators := make(chan *v1.ProcessIndicator)
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

func populateIndicatorFromCachedContainer(indicator *v1.ProcessIndicator, cachedContainer clusterentities.ContainerMetadata) {
	indicator.DeploymentId = cachedContainer.DeploymentID
	indicator.ContainerName = cachedContainer.ContainerName
	indicator.PodId = cachedContainer.PodID
}

// Process defines processes to process a ProcessIndicator
func (p *Pipeline) Process(signal *v1.ProcessSignal) {
	indicator := &v1.ProcessIndicator{
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
		p.indicators <- &v1.SensorEvent{
			Id:     indicator.GetId(),
			Action: v1.ResourceAction_CREATE_RESOURCE,
			Resource: &v1.SensorEvent_ProcessIndicator{
				ProcessIndicator: indicator,
			},
		}
	}
}
