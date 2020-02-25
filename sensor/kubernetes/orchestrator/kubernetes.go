package orchestrator

import (
	"context"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/orchestrator"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	coreV1Listers "k8s.io/client-go/listers/core/v1"
)

var (
	log = logging.LoggerForModule()
)

type kubernetesOrchestrator struct {
	client     *kubernetes.Clientset
	nodeLister coreV1Listers.NodeLister
}

// New returns a new kubernetes orchestrator client.
func New() orchestrator.Orchestrator {
	cs := client.MustCreateClientSet()
	sif := informers.NewSharedInformerFactory(cs, 0)
	nodeLister := sif.Core().V1().Nodes().Lister()
	sif.Start(context.Background().Done())

	return &kubernetesOrchestrator{
		client:     cs,
		nodeLister: nodeLister,
	}
}

func (k *kubernetesOrchestrator) GetNodeContainerRuntime(nodeName string) (string, error) {
	node, err := k.nodeLister.Get(nodeName)
	if err != nil {
		return "", err
	}
	return node.Status.NodeInfo.ContainerRuntimeVersion, nil
}
