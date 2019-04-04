package clusterstatus

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type nodeWatcher struct {
	stopC    <-chan struct{}
	updatesC chan<- *central.ClusterStatusUpdate

	deploymentEnvs *deploymentEnvSet
}

func newNodeWatcher(updatesC chan<- *central.ClusterStatusUpdate, stopC <-chan struct{}) *nodeWatcher {
	return &nodeWatcher{
		updatesC:       updatesC,
		stopC:          stopC,
		deploymentEnvs: newDeploymentEnvSet(),
	}
}

// Run starts watching for node events.
// CAVEAT: This function blocks until the stop channel that was passed upon construction is triggered.
func (w *nodeWatcher) Run(client *kubernetes.Clientset) {
	sif := informers.NewSharedInformerFactory(client, 0)
	nodeInformer := sif.Core().V1().Nodes().Informer()
	nodeInformer.AddEventHandler(w)
	nodeInformer.Run(w.stopC)
}

func (w *nodeWatcher) OnDelete(obj interface{}) {
	w.onChange(nil, obj)
}

func (w *nodeWatcher) OnUpdate(newObj, oldObj interface{}) {
	w.onChange(newObj, oldObj)
}

func (w *nodeWatcher) OnAdd(obj interface{}) {
	w.onChange(obj, nil)
}

func (w *nodeWatcher) onChange(newObj, oldObj interface{}) {
	var newDeploymentEnv, oldDeploymentEnv string

	if newObj != nil {
		newNode, _ := newObj.(*v1.Node)
		newDeploymentEnv = getDeploymentEnvFromNode(newNode)
	}
	if oldObj != nil {
		oldNode, _ := oldObj.(*v1.Node)
		oldDeploymentEnv = getDeploymentEnvFromNode(oldNode)
	}

	changed := w.deploymentEnvs.Replace(newDeploymentEnv, oldDeploymentEnv)
	if !changed {
		return
	}

	updateMsg := &central.ClusterStatusUpdate{
		Msg: &central.ClusterStatusUpdate_DeploymentEnvUpdate{
			DeploymentEnvUpdate: &central.DeploymentEnvironmentUpdate{
				Environments: w.deploymentEnvs.AsSlice(),
			},
		},
	}

	select {
	case w.updatesC <- updateMsg:
	case <-w.stopC:
	}
}
