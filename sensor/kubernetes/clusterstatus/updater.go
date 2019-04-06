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
	go u.run()
}

func (u *updaterImpl) sendMessage(msg *central.ClusterStatusUpdate) bool {
	select {
	case u.updates <- msg:
		return true
	case <-u.stopSig.Done():
		return false
	}
}

func (u *updaterImpl) run() {
	clusterMetadata := u.getClusterMetadata()
	cloudProviderMetadata := u.getCloudProviderMetadata()

	updateMessage := &central.ClusterStatusUpdate{
		Msg: &central.ClusterStatusUpdate_Status{
			Status: &storage.ClusterStatus{
				SensorVersion:        version.GetMainVersion(),
				ProviderMetadata:     cloudProviderMetadata,
				OrchestratorMetadata: clusterMetadata,
			},
		},
	}

	if !u.sendMessage(updateMessage) {
		return
	}

	deploymentEnvFromMD := getDeploymentEnvFromProviderMetadata(cloudProviderMetadata)

	// If we get the deployment environment from the cloud provider metadata, be happy with that - send the message
	// and just return.
	if deploymentEnvFromMD != "" {
		updateMessage := &central.ClusterStatusUpdate{
			Msg: &central.ClusterStatusUpdate_DeploymentEnvUpdate{
				DeploymentEnvUpdate: &central.DeploymentEnvironmentUpdate{
					Environments: []string{deploymentEnvFromMD},
				},
			},
		}

		u.sendMessage(updateMessage)
		return
	}

	// Otherwise, infer it from watching nodes.
	nw := newNodeWatcher(u.updates, u.stopSig.Done())
	nw.Run(u.client)
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
