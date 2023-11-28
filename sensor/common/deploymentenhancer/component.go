package deploymentenhancer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/booleanpolicy/networkpolicy"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/store"
)

var (
	log = logging.LoggerForModule()

	enhanceDeploymentQueueSize = 100
)

// CreateEnhancer .
func CreateEnhancer(provider store.Provider) common.SensorComponent {
	return &DeploymentEnhancer{
		responsesC:             make(chan *message.ExpiringMessage),
		enhanceDeploymentQueue: make(chan *central.DeploymentEnhancementRequest, enhanceDeploymentQueueSize),
		storeProvider:          provider,
	}
}

// DeploymentEnhancer .
type DeploymentEnhancer struct {
	responsesC             chan *message.ExpiringMessage
	enhanceDeploymentQueue chan *central.DeploymentEnhancementRequest
	storeProvider          store.Provider
}

// Notify .
func (d *DeploymentEnhancer) Notify(_ common.SensorComponentEvent) {}

// Start .
func (d *DeploymentEnhancer) Start() error {
	go func() {
		e, more := <-d.enhanceDeploymentQueue
		if !more {
			return
		}
		deployment := e.GetDeployment()

		localImages := set.NewStringSet()
		for _, c := range deployment.GetContainers() {
			imgName := c.GetImage().GetName()
			if d.storeProvider.Registries().IsLocal(imgName) {
				localImages.Add(imgName.GetFullName())
			}
		}

		permissionLevel := d.storeProvider.RBAC().GetPermissionLevelForDeployment(deployment)
		exposureInfo := d.storeProvider.Services().
			GetExposureInfos(deployment.GetNamespace(), deployment.GetPodLabels())

		deployment = d.storeProvider.Deployments().EnhanceDeploymentNoWrap(deployment, store.Dependencies{
			PermissionLevel: permissionLevel,
			Exposures:       exposureInfo,
			LocalImages:     localImages,
		})

		networkPolicies := d.storeProvider.NetworkPolicies().Find(deployment.GetNamespace(), deployment.GetPodLabels())
		appliedPolicies := networkpolicy.GenerateNetworkPoliciesAppliedObj(networkPolicies)

		// TODO: Handle context
		d.responsesC <- message.New(&central.MsgFromSensor{
			Msg: &central.MsgFromSensor_DeploymentEnhancementResponse{
				DeploymentEnhancementResponse: &central.DeploymentEnhancementResponse{
					Id:         e.GetId(),
					Deployment: deployment,
					NetworkPoliciesApplied: &central.DeploymentEnhancementResponseNetworkPoliciesApplied{
						HasEgressPolicy:  appliedPolicies.HasEgressNetworkPolicy,
						HasIngressPolicy: appliedPolicies.HasIngressNetworkPolicy,
						AppliedPolicies:  appliedPolicies.Policies,
					},
				},
			},
		})
	}()
	return nil
}

// Stop .
func (d *DeploymentEnhancer) Stop(_ error) {}

// Capabilities .
func (d *DeploymentEnhancer) Capabilities() []centralsensor.SensorCapability {
	// TODO: Add capability
	return nil
}

// ProcessMessage .
func (d *DeploymentEnhancer) ProcessMessage(msg *central.MsgToSensor) error {
	// TODO: Metrics
	enhanceReq := msg.GetDeploymentEnhancementRequest()
	if enhanceReq != nil {
		d.enhanceDeploymentQueue <- enhanceReq
	}
	return nil
}

// ResponsesC .
func (d *DeploymentEnhancer) ResponsesC() <-chan *message.ExpiringMessage {
	return d.responsesC
}
