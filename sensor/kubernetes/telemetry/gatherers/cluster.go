package gatherers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/providers"
	"github.com/stackrox/rox/pkg/telemetry"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/pkg/telemetry/gatherers"
	"github.com/stackrox/rox/sensor/common/store"
	"k8s.io/client-go/kubernetes"
)

// ClusterGatherer gathers cluster-related metrics
type ClusterGatherer struct {
	componentGatherer *gatherers.ComponentInfoGatherer
	nodeGatherer      *nodeGatherer
	namespaceGatherer *namespaceGatherer
	k8sClient         kubernetes.Interface
}

// NewClusterGatherer returns a new ClusterGatherer which will gather telemetry data about the cluster monitored by this
// sensor
func NewClusterGatherer(k8sClient kubernetes.Interface, deploymentStore store.DeploymentStore) *ClusterGatherer {
	return &ClusterGatherer{
		componentGatherer: gatherers.NewComponentInfoGatherer(),
		nodeGatherer:      newNodeGatherer(k8sClient),
		namespaceGatherer: newNamespaceGatherer(k8sClient, deploymentStore),
		k8sClient:         k8sClient,
	}
}

// Gather returns stats about the cluster this Sensor is monitoring
func (c *ClusterGatherer) Gather(ctx context.Context) *data.ClusterInfo {
	errorList := errorhelpers.NewErrorList("")

	orchestrator, err := c.getOrchestrator()
	errorList.AddError(err)

	providerMetadata := providers.GetMetadata(ctx)
	cloudProvider := telemetry.GetProviderString(providerMetadata)

	nodes, err := c.nodeGatherer.Gather(ctx)
	errorList.AddError(err)

	namespaces, nsErrors := c.namespaceGatherer.Gather(ctx)
	errorList.AddErrors(nsErrors...)

	return &data.ClusterInfo{
		Sensor: &data.SensorInfo{
			RoxComponentInfo:   c.componentGatherer.Gather(),
			CurrentlyConnected: true,
		},
		Orchestrator:  orchestrator,
		Nodes:         nodes,
		Namespaces:    namespaces,
		CloudProvider: cloudProvider,
		Errors:        errorList.ErrorStrings(),
	}
}

func (c *ClusterGatherer) getOrchestrator() (*data.OrchestratorInfo, error) {
	serverVersion, err := c.k8sClient.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	orchestrator := storage.ClusterType_KUBERNETES_CLUSTER.String()
	if env.OpenshiftAPI.BooleanSetting() {
		orchestrator = storage.ClusterType_OPENSHIFT_CLUSTER.String()
	}
	return &data.OrchestratorInfo{
		Orchestrator:        orchestrator,
		OrchestratorVersion: serverVersion.GitVersion,
	}, nil
}
