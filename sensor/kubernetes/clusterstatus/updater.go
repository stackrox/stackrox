package clusterstatus

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/providers"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common/clusterstatus"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logging.LoggerForModule()
)

type updaterImpl struct {
	client *kubernetes.Clientset

	updates chan *central.ClusterStatusUpdate
	stopSig concurrency.Signal

	deploymentEnvs *deploymentEnvSet
}

func (u *updaterImpl) Start() {
	sif := informers.NewSharedInformerFactory(u.client, 0)
	nodeInformer := sif.Core().V1().Nodes().Informer()
	nodeInformer.AddEventHandler(u)
	go nodeInformer.Run(u.stopSig.Done())
	go u.run()
}

func (u *updaterImpl) OnDelete(obj interface{}) {
	u.onChange(nil, obj)
}

func (u *updaterImpl) OnUpdate(newObj, oldObj interface{}) {
	u.onChange(newObj, oldObj)
}

func (u *updaterImpl) OnAdd(obj interface{}) {
	u.onChange(obj, nil)
}

func (u *updaterImpl) onChange(newObj, oldObj interface{}) {
	var newDeploymentEnv, oldDeploymentEnv string

	if newObj != nil {
		newNode, _ := newObj.(*v1.Node)
		newDeploymentEnv = getDeploymentEnvironment(newNode)
	}
	if oldObj != nil {
		oldNode, _ := oldObj.(*v1.Node)
		oldDeploymentEnv = getDeploymentEnvironment(oldNode)
	}

	changed := u.deploymentEnvs.Replace(newDeploymentEnv, oldDeploymentEnv)
	if !changed {
		return
	}

	u.sendDeploymentEnvUpdateMessage()
}

func (u *updaterImpl) sendDeploymentEnvUpdateMessage() {
	environments := u.deploymentEnvs.AsSlice()

	msg := &central.ClusterStatusUpdate{
		Msg: &central.ClusterStatusUpdate_DeploymentEnvUpdate{
			DeploymentEnvUpdate: &central.DeploymentEnvironmentUpdate{
				Environments: environments,
			},
		},
	}

	select {
	case u.updates <- msg:
	case <-u.stopSig.Done():
	}
}

func (u *updaterImpl) run() {
	updateMessage := &central.ClusterStatusUpdate{
		Msg: &central.ClusterStatusUpdate_Status{
			Status: &storage.ClusterStatus{
				SensorVersion:        version.GetMainVersion(),
				ProviderMetadata:     u.getCloudProviderMetadata(),
				OrchestratorMetadata: u.getClusterMetadata(),
			},
		},
	}
	select {
	case u.updates <- updateMessage:
	case <-u.stopSig.Done():
	}
}

func (u *updaterImpl) Stop() {
	u.stopSig.Signal()
}

func (u *updaterImpl) Updates() <-chan *central.ClusterStatusUpdate {
	return u.updates
}

func (u *updaterImpl) getClusterMetadata() *storage.OrchestratorMetadata {
	serverVersion, err := u.client.ServerVersion()
	if err != nil {
		log.Errorf("Could not get cluster metadata: %v", err)
		return nil
	}

	buildDate, err := time.Parse(time.RFC3339, serverVersion.BuildDate)
	if err != nil {
		log.Error(err)
	}

	return &storage.OrchestratorMetadata{
		Version:   serverVersion.GitVersion,
		BuildDate: protoconv.ConvertTimeToTimestamp(buildDate),
	}
}

func (u *updaterImpl) getCloudProviderMetadata() *storage.ProviderMetadata {
	m := providers.GetMetadata()
	if m == nil {
		log.Infof("No Cloud Provider metadata is found")
	}
	return m
}

// NewUpdater returns a new ready-to-use updater.
func NewUpdater() clusterstatus.Updater {
	return &updaterImpl{
		client:         client.MustCreateClientSet(),
		updates:        make(chan *central.ClusterStatusUpdate),
		stopSig:        concurrency.NewSignal(),
		deploymentEnvs: newDeploymentEnvSet(),
	}
}
