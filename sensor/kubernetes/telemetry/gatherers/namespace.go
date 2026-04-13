package gatherers

import (
	"context"

	"github.com/stackrox/rox/pkg/telemetry"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

type namespaceGatherer struct {
	dynClient       dynamic.Interface
	deploymentStore store.DeploymentStore
}

func newNamespaceGatherer(dynClient dynamic.Interface, deploymentStore store.DeploymentStore) *namespaceGatherer {
	return &namespaceGatherer{
		dynClient:       dynClient,
		deploymentStore: deploymentStore,
	}
}

// Gather returns a list of stats about all the namespaces in the cluster this Sensor is monitoring
func (n *namespaceGatherer) Gather(ctx context.Context) ([]*data.NamespaceInfo, []error) {
	var errList []error
	namespaceList, err := n.dynClient.Resource(client.NamespaceGVR).List(ctx, v1.ListOptions{})
	if err != nil {
		errList = append(errList, err)
		return nil, errList
	}

	namespaceInfoList := make([]*data.NamespaceInfo, 0, len(namespaceList.Items))
	for _, namespace := range namespaceList.Items {
		nsName := namespace.GetName()
		podList, err := n.dynClient.Resource(client.PodGVR).Namespace(nsName).List(ctx, v1.ListOptions{})
		if err != nil {
			errList = append(errList, err)
			continue
		}

		name := nsName
		if !telemetry.WellKnownNamespaces.Contains(name) {
			name = ""
		}
		namespaceInfoList = append(namespaceInfoList, &data.NamespaceInfo{
			ID:             string(namespace.GetUID()),
			Name:           name,
			NumPods:        len(podList.Items),
			NumDeployments: n.deploymentStore.CountDeploymentsForNamespace(nsName),
		})
	}
	return namespaceInfoList, errList
}
