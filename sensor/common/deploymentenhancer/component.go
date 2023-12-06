package deploymentenhancer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
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
	log.Debugf("Received message to process in DeploymentEnhancer: %v++", msg)
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
			deployments := d.extractAndEnrichDeployments(deploymentMsg)
			d.sendDeploymentsToCentral(requestID, deployments)
		}
	}()
	return nil
}

func (d *DeploymentEnhancer) extractAndEnrichDeployments(deploymentMsg *central.DeploymentEnhancementRequest) []*storage.Deployment {
	log.Debugf("Received deploymentEnhancement msg with %v deployment(s)", len(deploymentMsg.GetMsg().GetDeployments()))

	deployments := deploymentMsg.GetMsg().GetDeployments()
	if deployments == nil {
		log.Warnf("received deploymentEnhancement msg with no deployments")
	}

	var ret []*storage.Deployment

	for _, deployment := range deployments {
		enriched, err := d.enrichDeployment(deployment)
		if err != nil {
			log.Warnf("Failed to enrich deployment: %v", deployment)
			continue
		}
		ret = append(ret, enriched)
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

func (d *DeploymentEnhancer) enrichDeployment(deployment *storage.Deployment) (*storage.Deployment, error) {
	localImages := set.NewStringSet()
	for _, c := range deployment.GetContainers() {
		imgName := c.GetImage().GetName()
		if d.storeProvider.Registries().IsLocal(imgName) {
			localImages.Add(imgName.GetFullName())
		}
	}

	p := d.storeProvider.RBAC().GetPermissionLevelForDeployment(deployment)
	e := d.storeProvider.Services().GetExposureInfos(deployment.GetNamespace(), deployment.GetPodLabels())

	deployment = d.storeProvider.Deployments().EnhanceDeploymentReadOnly(deployment, store.Dependencies{
		PermissionLevel: p,
		Exposures:       e,
		LocalImages:     localImages,
	})

	return deployment, nil
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

// Notify .
func (d *DeploymentEnhancer) Notify(_ common.SensorComponentEvent) {}
