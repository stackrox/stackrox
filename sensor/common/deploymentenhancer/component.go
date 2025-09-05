package deploymentenhancer

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sensor/queue"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/store"
)

var (
	log = logging.LoggerForModule()
)

// The DeploymentEnhancer takes a list of Deployments and enhances them with all available information
type DeploymentEnhancer struct {
	responsesC       chan *message.ExpiringMessage
	deploymentsQueue chan *central.DeploymentEnhancementRequest
	storeProvider    store.Provider
	ctx              context.Context
	ctxCancel        context.CancelFunc
}

func (d *DeploymentEnhancer) Name() string {
	return "deploymentenhancer.DeploymentEnhancer"
}

// CreateEnhancer creates a new Enhancer
func CreateEnhancer(provider store.Provider) common.SensorComponent {
	ctx, ctxCancel := context.WithCancel(context.Background())

	return &DeploymentEnhancer{
		responsesC:       make(chan *message.ExpiringMessage),
		deploymentsQueue: make(chan *central.DeploymentEnhancementRequest, queue.ScaleSizeOnNonDefault(env.SensorDeploymentEnhancementQueueSize)),
		storeProvider:    provider,
		ctx:              ctx,
		ctxCancel:        ctxCancel,
	}
}

func (d *DeploymentEnhancer) Filter(msg *central.MsgToSensor) bool {
	return msg.GetDeploymentEnhancementRequest() != nil
}

// ProcessMessage takes an incoming message and queues it for enhancement
func (d *DeploymentEnhancer) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	toEnhance := msg.GetDeploymentEnhancementRequest()
	if toEnhance == nil {
		return nil
	}
	if toEnhance.GetMsg() == nil {
		return errox.ReferencedObjectNotFound.New("received empty message")
	}
	log.Debugf("Received message to process in DeploymentEnhancer: %+v", toEnhance)

	select {
	case d.deploymentsQueue <- toEnhance:
		metrics.IncrementDeploymentEnhancerQueueSize()
		return nil
	default:
		return errox.ResourceExhausted.Newf("DeploymentEnhancer queue has reached its limit of %d", len(d.deploymentsQueue))
	}
}

// Start starts the component
func (d *DeploymentEnhancer) Start() error {
	go func() {
		for {
			select {
			case <-d.ctx.Done():
				return
			case deploymentMsg, more := <-d.deploymentsQueue:
				if !more {
					return
				}
				metrics.DecrementDeploymentEnhancerQueueSize()
				if deploymentMsg.GetMsg() == nil {
					log.Warnf("Received empty deploymentEnhancement message. Discarding request.")
					return
				}
				requestID := deploymentMsg.GetMsg().GetId()
				if requestID == "" {
					log.Warnf("Received deploymentEnhancement msg with empty request ID. Discarding request.")
					continue
				}
				d.sendDeploymentsToCentral(requestID, d.enhanceDeployments(deploymentMsg))
			}
		}
	}()
	return nil
}

func (d *DeploymentEnhancer) enhanceDeployments(deploymentMsg *central.DeploymentEnhancementRequest) []*storage.Deployment {
	var ret []*storage.Deployment

	if deploymentMsg.GetMsg() == nil || deploymentMsg.GetMsg().GetDeployments() == nil {
		log.Warnf("Received empty deploymentEnhancement message")
		return ret
	}

	deployments := deploymentMsg.GetMsg().GetDeployments()

	log.Debugf("Received deploymentEnhancement message with %d deployment(s)", len(deployments))
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
	if features.ClusterAwareDeploymentCheck.Enabled() {
		return []centralsensor.SensorCapability{centralsensor.SensorEnhancedDeploymentCheckCap}
	}
	return nil
}

// ResponsesC returns the response channel of this component
func (d *DeploymentEnhancer) ResponsesC() <-chan *message.ExpiringMessage {
	return d.responsesC
}

// Stop stops the component
func (d *DeploymentEnhancer) Stop() {
	d.ctxCancel()
}

// Notify is unimplemented, part of the common interface
func (d *DeploymentEnhancer) Notify(_ common.SensorComponentEvent) {}
