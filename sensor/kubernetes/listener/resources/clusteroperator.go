package resources

import (
	v1 "github.com/openshift/api/config/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/orchestratornamespaces"
)

// clusterOperatorDispatcher handles cluster operator events
type clusterOperatorDispatcher struct {
	orchestratorNamespaces *orchestratornamespaces.OrchestratorNamespaces
}

// newClusterOperatorDispatcher creates and returns a new cluster operator dispatcher.
func newClusterOperatorDispatcher(namespaces *orchestratornamespaces.OrchestratorNamespaces) *clusterOperatorDispatcher {
	return &clusterOperatorDispatcher{
		orchestratorNamespaces: namespaces,
	}
}

// ProcessEvent processes a cluster operator resource event, and returns the sensor events to emit in response.
func (c *clusterOperatorDispatcher) ProcessEvent(obj, _ interface{}, _ central.ResourceAction) *component.ResourceEvent {
	clusterOperator, ok := obj.(*v1.ClusterOperator)

	if !ok {
		return nil
	}

	/*
	  Sample RelatedObject:
	  relatedObjects:
	  - group: ""
	    name: openshift-machine-api
	    resource: namespaces
	  - group: machine.openshift.io
	    name: ""
	    namespace: openshift-machine-api
	    resource: machines
	*/
	for _, obj := range clusterOperator.Status.RelatedObjects {
		if obj.Resource == "namespaces" {
			log.Debugf("Adding namespace %s to orchestrator namespace map", obj.Name)
			c.orchestratorNamespaces.AddNamespace(obj.Name)
		}
	}
	return nil
}
