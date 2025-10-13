package gatherers

import (
	"context"

	"github.com/stackrox/rox/pkg/telemetry"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/sensor/common/store"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type namespaceGatherer struct {
	k8sClient       kubernetes.Interface
	deploymentStore store.DeploymentStore
}

func newNamespaceGatherer(k8sClient kubernetes.Interface, deploymentStore store.DeploymentStore) *namespaceGatherer {
	return &namespaceGatherer{
		k8sClient:       k8sClient,
		deploymentStore: deploymentStore,
	}
}

// Gather returns a list of stats about all the namespaces in the cluster this Sensor is monitoring
func (n *namespaceGatherer) Gather(ctx context.Context) ([]*data.NamespaceInfo, []error) {
	var errList []error
	namespaceList, err := n.k8sClient.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	if err != nil {
		errList = append(errList, err)
		return nil, errList
	}

	namespaceInfoList := make([]*data.NamespaceInfo, 0, len(namespaceList.Items))
	for _, namespace := range namespaceList.Items {
		podsForNamespace := n.k8sClient.CoreV1().Pods(namespace.Name)
		pods, err := podsForNamespace.List(ctx, v1.ListOptions{})
		if err != nil {
			errList = append(errList, err)
			continue
		}

		name := namespace.GetName()
		if !telemetry.WellKnownNamespaces.Contains(name) {
			name = ""
		}
		namespaceInfoList = append(namespaceInfoList, &data.NamespaceInfo{
			ID:             string(namespace.GetUID()),
			Name:           name,
			NumPods:        len(pods.Items),
			NumDeployments: n.deploymentStore.CountDeploymentsForNamespace(namespace.GetName()),
		})
	}
	return namespaceInfoList, errList
}
