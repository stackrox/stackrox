package deploymentenhancer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/store"
)

var (
	log                 = logging.LoggerForModule()
	deploymentQueueSize = 50 // TODO(ROX-21291): Configurable via env var
)

// The DeploymentEnhancer takes a list of Deployments and enhances them with all available information
type DeploymentEnhancer struct {
	responsesC       chan *message.ExpiringMessage
	deploymentsQueue chan *central.DeploymentEnhancementRequest
	storeProvider    store.Provider
}

// CreateEnhancer creates a new Enhancer
func CreateEnhancer(provider store.Provider) common.SensorComponent {
	return &DeploymentEnhancer{
		responsesC:       make(chan *message.ExpiringMessage),
		deploymentsQueue: make(chan *central.DeploymentEnhancementRequest, deploymentQueueSize),
		storeProvider:    provider,
	}
}

// ProcessMessage takes an incoming message and queues it for enhancement
func (d *DeploymentEnhancer) ProcessMessage(msg *central.MsgToSensor) error {
	toEnhance := msg.GetDeploymentEnhancementRequest()
	if toEnhance == nil {
		return nil
	}
	log.Debugf("Received message to process in DeploymentEnhancer: %+v", toEnhance)
	d.deploymentsQueue <- toEnhance
	return nil
}

// Start starts the component
func (d *DeploymentEnhancer) Start() error {
	go func() {
		for {
			deploymentMsg, more := <-d.deploymentsQueue
			if !more {
				return
			}
			requestID := deploymentMsg.GetMsg().GetId()
			if requestID == "" {
				log.Warnf("received deploymentEnhancement msg with empty request ID. Discarding request.")
				continue
			}
			deployments := d.enhanceDeployments(deploymentMsg)
			d.sendDeploymentsToCentral(requestID, deployments)
		}
	}()
	return nil
}

func (d *DeploymentEnhancer) enhanceDeployments(deploymentMsg *central.DeploymentEnhancementRequest) []*storage.Deployment {
	var ret []*storage.Deployment

	deployments := deploymentMsg.GetMsg().GetDeployments()
	if deployments == nil {
		log.Warnf("received deploymentEnhancement message with no deployments")
		return ret
	}

	log.Debugf("Received deploymentEnhancement msg with %d deployment(s)", len(deploymentMsg.GetMsg().GetDeployments()))
	for _, deployment := range deployments {
		d.enhanceDeployment(deployment)
		ret = append(ret, deployment)
	}

	return ret
}

func (d *DeploymentEnhancer) sendDeploymentsToCentral(id string, deployments []*storage.Deployment) {
	d.responsesC <- message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_DeploymentEnhancementResponse{
			DeploymentEnhancementResponse: &central.DeploymentEnhancementResponse{
				Msg: &central.DeploymentEnhancementMessage{
					Id:          id,
					Deployments: deployments,
				},
			},
		},
	})
}

func (d *DeploymentEnhancer) enhanceDeployment(deployment *storage.Deployment) {
	d.storeProvider.Deployments().EnhanceDeploymentReadOnly(deployment, store.Dependencies{
		PermissionLevel: d.storeProvider.RBAC().GetPermissionLevelForDeployment(deployment),
		Exposures:       d.storeProvider.Services().GetExposureInfos(deployment.GetNamespace(), deployment.GetPodLabels()),
	})
}

// Capabilities return the capabilities of this component
func (d *DeploymentEnhancer) Capabilities() []centralsensor.SensorCapability {
	// TODO(ROX-21197): Add Capability
	return nil
}

// ResponsesC returns the response channel of this component
func (d *DeploymentEnhancer) ResponsesC() <-chan *message.ExpiringMessage {
	return d.responsesC
}

// Stop stops the component
func (d *DeploymentEnhancer) Stop(_ error) {
	defer close(d.deploymentsQueue)
}

// Notify is unimplemented, part of the common interface
func (d *DeploymentEnhancer) Notify(_ common.SensorComponentEvent) {}
