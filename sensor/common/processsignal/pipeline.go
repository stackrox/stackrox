package processsignal

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/process/filter"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stackrox/stackrox/sensor/common/clusterentities"
	"github.com/stackrox/stackrox/sensor/common/detector"
	"github.com/stackrox/stackrox/sensor/common/metrics"
)

var (
	log = logging.LoggerForModule()
)

// Pipeline is the struct that handles a process signal
type Pipeline struct {
	clusterEntities    *clusterentities.Store
	indicators         chan *central.MsgFromSensor
	enrichedIndicators chan *storage.ProcessIndicator
	enricher           *enricher
	processFilter      filter.Filter
	detector           detector.Detector
}

// NewProcessPipeline defines how to process a ProcessIndicator
func NewProcessPipeline(indicators chan *central.MsgFromSensor, clusterEntities *clusterentities.Store, processFilter filter.Filter, detector detector.Detector) *Pipeline {
	enrichedIndicators := make(chan *storage.ProcessIndicator)
	p := &Pipeline{
		clusterEntities:    clusterEntities,
		indicators:         indicators,
		enricher:           newEnricher(clusterEntities, enrichedIndicators),
		enrichedIndicators: enrichedIndicators,
		processFilter:      processFilter,
		detector:           detector,
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
		if !p.processFilter.Add(indicator) {
			continue
		}
		p.detector.ProcessIndicator(indicator)

		p.indicators <- &central.MsgFromSensor{Msg: &central.MsgFromSensor_Event{Event: &central.SensorEvent{
			Id:     indicator.GetId(),
			Action: central.ResourceAction_CREATE_RESOURCE,
			Resource: &central.SensorEvent_ProcessIndicator{
				ProcessIndicator: indicator,
			},
		},
		},
		}
	}
}
