package processsignal

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/clusterentities"
)

const (
	signalRetries       = 30
	signalRetryInterval = 2 * time.Second
)

var logger = logging.LoggerForModule()

// Pipeline is the struct that handles a process signal
type Pipeline struct {
	clusterEntities *clusterentities.Store
	indicators      chan *v1.SensorEvent
}

// NewProcessPipeline defines how to process a ProcessIndicator
func NewProcessPipeline(indicators chan *v1.SensorEvent, clusterEntities *clusterentities.Store) *Pipeline {
	return &Pipeline{
		clusterEntities: clusterEntities,
		indicators:      indicators,
	}
}

func (p *Pipeline) reprocessSignalLater(indicator *v1.ProcessIndicator) {
	t := time.NewTicker(signalRetryInterval)
	logger.Infof("Trying to reprocess '%s'", indicator.GetSignal().GetExecFilePath())
	for i := 0; i < signalRetries; i++ {
		<-t.C
		deploymentID := p.clusterEntities.LookupByContainerID(indicator.GetSignal().GetContainerId())
		if deploymentID != "" {
			indicator.DeploymentId = deploymentID
			p.sendIndicatorEvent(indicator)
			return
		}
	}
	logger.Errorf("Dropping this on the floor: %s", proto.MarshalTextString(indicator))
}

// Process defines processes to process a ProcessIndicator
func (p *Pipeline) Process(signal *v1.ProcessSignal) {
	indicator := &v1.ProcessIndicator{
		Id:     uuid.NewV4().String(),
		Signal: signal,
	}

	// indicator.GetSignal() is never nil at this point
	deploymentID := p.clusterEntities.LookupByContainerID(indicator.GetSignal().GetContainerId())
	if deploymentID == "" {
		go p.reprocessSignalLater(indicator)
		return
	}
	indicator.DeploymentId = deploymentID
	p.sendIndicatorEvent(indicator)
}

func (p *Pipeline) sendIndicatorEvent(indicator *v1.ProcessIndicator) {
	p.indicators <- &v1.SensorEvent{
		Id:     indicator.GetId(),
		Action: v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_ProcessIndicator{
			ProcessIndicator: indicator,
		},
	}
}
