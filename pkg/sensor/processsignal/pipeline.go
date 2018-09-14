package processsignal

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sensor/cache"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	signalRetries       = 10
	signalRetryInterval = 2 * time.Second
)

var logger = logging.LoggerForModule()

// Pipeline is the struct that handles a process signal
type Pipeline struct {
	pendingCache *cache.PendingEvents
	indicators   chan *listeners.EventWrap
}

// NewProcessPipeline defines how to process a ProcessIndicator
func NewProcessPipeline(indicators chan *listeners.EventWrap, pendingCache *cache.PendingEvents) *Pipeline {
	return &Pipeline{
		pendingCache: pendingCache,
		indicators:   indicators,
	}
}

func (p *Pipeline) reprocessSignalLater(indicator *v1.ProcessIndicator) {
	t := time.NewTicker(signalRetryInterval)
	logger.Infof("Trying to reprocess '%s'", indicator.GetSignal().GetExecFilePath())
	for i := 0; i < signalRetries; i++ {
		<-t.C
		deploymentID, exists := p.pendingCache.FetchDeploymentByContainer(indicator.GetSignal().GetContainerId())
		if exists {
			indicator.DeploymentId = deploymentID
			p.wrapAndSendIndicator(indicator)
			return
		}
	}
	logger.Errorf("Dropping this on the floor: %+v", proto.MarshalTextString(indicator))
}

// Process defines processes to process a ProcessIndicator
func (p *Pipeline) Process(signal *v1.ProcessSignal) {
	indicator := &v1.ProcessIndicator{
		Id:     uuid.NewV4().String(),
		Signal: signal,
	}

	// indicator.GetSignal() is never nil at this point
	deploymentID, exists := p.pendingCache.FetchDeploymentByContainer(indicator.GetSignal().GetContainerId())
	if !exists {
		go p.reprocessSignalLater(indicator)
		return
	}
	indicator.DeploymentId = deploymentID
	p.wrapAndSendIndicator(indicator)
}

func (p *Pipeline) wrapAndSendIndicator(indicator *v1.ProcessIndicator) {
	eventWrap := &listeners.EventWrap{
		SensorEvent: &v1.SensorEvent{
			Id:     indicator.GetId(),
			Action: v1.ResourceAction_CREATE_RESOURCE,
			Resource: &v1.SensorEvent_ProcessIndicator{
				ProcessIndicator: indicator,
			},
		},
	}
	p.indicators <- eventWrap
}
