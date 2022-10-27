package resolver

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/message"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output"
)

var (
	log = logging.LoggerForModule()
)

type DeploymentResolver interface {
	ProcessDependencies(ref message.DeploymentRef) ([]*storage.Deployment, error)
}

type Resolver interface {
	Send(pipelineMessage *message.ResourceEvent)
	SetDeploymentResolver(resolver DeploymentResolver)
}

type resolverImpl struct {
	innerQueue                   chan *message.ResourceEvent
	outputQueue                  output.Queue
	deploymentDependencyResolver DeploymentResolver
}

func New(queue output.Queue) Resolver {
	resolver := &resolverImpl{
		outputQueue: queue,
		innerQueue:  make(chan *message.ResourceEvent, 100),
	}
	go resolver.startProcessing()
	return resolver
}

func (r *resolverImpl) SetDeploymentResolver(resolver DeploymentResolver) {
	r.deploymentDependencyResolver = resolver
}

func (r *resolverImpl) startProcessing() {
	for {
		// TODO: use a select block + signal to abort
		msg, more := <-r.innerQueue
		if !more {
			return
		}
		r.processMessage(msg)
	}
}

func (r *resolverImpl) processMessage(pipelineMessage *message.ResourceEvent) {
	// log.Infof("PROCESSING DEPLOYMENT IDS: %v", pipelineMessage.DeploymentRefs)
	for _, deploymentRef := range pipelineMessage.DeploymentRefs {
		// Process deployments
		// This should be executed whenever a deployment event was triggered
		// or if a dependency from a deployment was changed
		// For dependencies, multiple deployments could have the resource as a
		// dependency, this is why we need to process each one individually.
		deployments, err := r.deploymentDependencyResolver.ProcessDependencies(deploymentRef)
		if err != nil {
			log.Warnf("deployment resolver failed for deployments ref %s: %s", deploymentRef, err)
			continue
		}

		for _, deployment := range deployments {
			pipelineMessage.CompatibilityDetectionDeployment = append(pipelineMessage.CompatibilityDetectionDeployment,
				message.CompatibilityDetectionMessage{
					Object: deployment,
					Action: deploymentRef.Action,
				})

			pipelineMessage.ForwardMessages = append(pipelineMessage.ForwardMessages,
				&central.SensorEvent{
					Id:     deploymentRef.Id,
					Action: deploymentRef.Action,
					Resource: &central.SensorEvent_Deployment{
						Deployment: deployment,
					},
				})
		}
	}

	// Clean up deployment ref field
	pipelineMessage.DeploymentRefs = []message.DeploymentRef{}

	r.outputQueue.Send(pipelineMessage)
}

func (r *resolverImpl) Send(pipelineMessage *message.ResourceEvent) {
	r.innerQueue <- pipelineMessage
}
