package clusterstatus

import (
	"context"
	"sort"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/deploymentenvs"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/providers"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logging.LoggerForModule()
)

type updaterImpl struct {
	client kubernetes.Interface

	updates chan *central.MsgFromSensor
	stopSig concurrency.Signal
}

func (u *updaterImpl) Start() error {
	go u.run()
	return nil
}

func (u *updaterImpl) Stop(_ error) {
	u.stopSig.Signal()
}

func (u *updaterImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (u *updaterImpl) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (u *updaterImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return u.updates
}

func (u *updaterImpl) sendMessage(msg *central.ClusterStatusUpdate) bool {
	select {
	case u.updates <- &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ClusterStatusUpdate{
			ClusterStatusUpdate: msg,
		},
	}:
		return true
	case <-u.stopSig.Done():
		return false
	}
}

func (u *updaterImpl) run() {
	clusterMetadata := u.getClusterMetadata()
	cloudProviderMetadata := u.getCloudProviderMetadata(context.Background())

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

	deploymentEnvFromMD := deploymentenvs.GetDeploymentEnvFromProviderMetadata(cloudProviderMetadata)

	// If we don't get any deployment environment info from the cloud provider metadata, just return - there's nothing left for us to do.
	if deploymentEnvFromMD == "" {
		return
	}
	updateMessage = &central.ClusterStatusUpdate{
		Msg: &central.ClusterStatusUpdate_DeploymentEnvUpdate{
			DeploymentEnvUpdate: &central.DeploymentEnvironmentUpdate{
				Environments: []string{deploymentEnvFromMD},
			},
		},
	}

	u.sendMessage(updateMessage)
}

func (u *updaterImpl) getClusterMetadata() *storage.OrchestratorMetadata {
	serverVersion, err := u.client.Discovery().ServerVersion()
	if err != nil {
		log.Errorf("Could not get cluster metadata: %v", err)
		return nil
	}

	buildDate, err := time.Parse(time.RFC3339, serverVersion.BuildDate)
	if err != nil {
		log.Error(err)
	}

	return &storage.OrchestratorMetadata{
		Version:     serverVersion.GitVersion,
		BuildDate:   protoconv.ConvertTimeToTimestamp(buildDate),
		ApiVersions: u.getAPIVersions(),
	}
}

// API versions exists as the fields in the kube client.
func (u *updaterImpl) getAPIVersions() []string {
	groupList, err := u.client.Discovery().ServerGroups()
	if err != nil {
		log.Errorf("unable to fetch api-versions: %s", err)
		return nil
	}

	apiVersions := metav1.ExtractGroupVersions(groupList)
	sort.Strings(apiVersions)
	return apiVersions
}

func (u *updaterImpl) getCloudProviderMetadata(ctx context.Context) *storage.ProviderMetadata {
	m := providers.GetMetadata(ctx)
	if m == nil {
		log.Info("No Cloud Provider metadata is found")
	}
	return m
}

// NewUpdater returns a new ready-to-use updater.
func NewUpdater(client kubernetes.Interface) common.SensorComponent {
	return &updaterImpl{
		client:  client,
		updates: make(chan *central.MsgFromSensor),
		stopSig: concurrency.NewSignal(),
	}
}
