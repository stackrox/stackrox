package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	v1 "k8s.io/api/batch/v1"
)

// jobDispatcherImpl implements the Dispatcher interface and handles the Job lifecycle then pushes it to
// the generic deploymentDispatcherImpl. Namely, Jobs are considered removed once they have been completed
type jobDispatcherImpl struct {
	removedCached        set.StringSet
	deploymentDispatcher Dispatcher
}

// newDeploymentDispatcher creates and returns a new deployment dispatcher instance.
func newJobDispatcherImpl(handler *deploymentHandler) Dispatcher {
	return &jobDispatcherImpl{
		removedCached:        set.NewStringSet(),
		deploymentDispatcher: newDeploymentDispatcher(kubernetes.Job, handler),
	}
}

// ProcessEvent processes a deployment resource events, and returns the sensor events to emit in response.
func (d *jobDispatcherImpl) ProcessEvent(obj, oldObj interface{}, action central.ResourceAction) *component.ResourceEvent {
	job, ok := obj.(*v1.Job)
	if !ok {
		log.Errorf("could not process object because it is not of type job: %T", obj)
		return nil
	}

	// If we have already sent 1 remove for this job, then do not send another
	if action == central.ResourceAction_REMOVE_RESOURCE && d.removedCached.Remove(string(job.GetUID())) {
		return nil
	}

	if job.Status.CompletionTime != nil && action != central.ResourceAction_REMOVE_RESOURCE {
		// If we have already sent 1 remove for this job, then do not send another
		if !d.removedCached.Add(string(job.GetUID())) {
			return nil
		}
		log.Debugf("Job %s is completed and is being marked as removed", job.Name)
		return d.deploymentDispatcher.ProcessEvent(obj, oldObj, central.ResourceAction_REMOVE_RESOURCE)
	}

	return d.deploymentDispatcher.ProcessEvent(obj, oldObj, action)
}
