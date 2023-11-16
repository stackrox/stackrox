package clusterstatus

import (
	"context"
	"encoding/json"
	"sort"
	"sync/atomic"
	"time"

	v1 "github.com/openshift/api/config/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/deploymentenvs"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/providers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
)

var (
	log                  = logging.LoggerForModule()
	getVersionRetryDelay = 3 * time.Second
	getVersionRetries    = 3
	getVersionTimeout    = 10 * time.Second
)

type updaterImpl struct {
	client     client.Interface
	kubeClient kubernetes.Interface

	updates chan *message.ExpiringMessage
	stopSig concurrency.Signal

	offlineMode   *atomic.Bool
	context       context.Context
	contextMtx    sync.Mutex
	cancelContext context.CancelFunc
	// This function is needed to be able to mock in test
	getProviders func(context.Context) *storage.ProviderMetadata
}

func (u *updaterImpl) Start() error {
	// We don't do anything on Start, run will be called when Central is reachable.
	return nil
}

func (u *updaterImpl) Stop(_ error) {
	u.stopSig.Signal()
}

func (u *updaterImpl) Notify(e common.SensorComponentEvent) {
	switch e {
	case common.SensorComponentEventCentralReachable:
		if u.offlineMode.CompareAndSwap(true, false) {
			u.createContext()
			go u.run()
		}
	case common.SensorComponentEventOfflineMode:
		if u.offlineMode.CompareAndSwap(false, true) {
			u.cancelCurrentContext()
		}
	}
}

func (u *updaterImpl) cancelCurrentContext() {
	u.contextMtx.Lock()
	defer u.contextMtx.Unlock()
	if u.cancelContext != nil {
		u.cancelContext()
	}
}

func (u *updaterImpl) createContext() {
	u.contextMtx.Lock()
	defer u.contextMtx.Unlock()
	u.context, u.cancelContext = context.WithCancel(context.Background())
}

func (u *updaterImpl) getCurrentContext() context.Context {
	u.contextMtx.Lock()
	defer u.contextMtx.Unlock()
	return u.context
}

func (u *updaterImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (u *updaterImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (u *updaterImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return u.updates
}

func (u *updaterImpl) sendMessage(msg *central.ClusterStatusUpdate) bool {
	ctx := u.getCurrentContext()
	select {
	case <-ctx.Done():
		return false
	case u.updates <- message.NewExpiring(ctx, &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ClusterStatusUpdate{
			ClusterStatusUpdate: msg,
		},
	}):
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
	serverVersion, err := u.kubeClient.Discovery().ServerVersion()
	if err != nil {
		log.Errorf("Could not get cluster metadata: %v", err)
		return nil
	}

	buildDate, err := time.Parse(time.RFC3339, serverVersion.BuildDate)
	if err != nil {
		log.Error(err)
	}

	metadata := &storage.OrchestratorMetadata{
		Version:     serverVersion.GitVersion,
		BuildDate:   protoconv.ConvertTimeToTimestamp(buildDate),
		ApiVersions: u.getAPIVersions(),
	}

	if env.OpenshiftAPI.BooleanSetting() {
		// Update Openshift version
		openshiftVersion, err := u.getOpenshiftVersion()
		if kerrors.IsNotFound(err) {
			// Try legacy way to get version for Openshift 3.11
			log.Info("Cannot get Openshift version from operator, trying to get it through legacy API")
			openshiftVersion, err = u.getOpenshiftVersionLegacyAPI()
		}
		if err != nil {
			if kerrors.IsForbidden(err) || kerrors.IsUnauthorized(err) {
				log.Errorf("OpenShift version not found (must be logged in to cluster as admin): %v", err)
			} else {
				log.Errorf("Fail to get Openshift version: %v", err)
			}
			return metadata
		}
		log.Infof("Openshift version: %s", openshiftVersion)
		metadata.IsOpenshift = &storage.OrchestratorMetadata_OpenshiftVersion{OpenshiftVersion: openshiftVersion}
	}
	return metadata
}

func (u *updaterImpl) getOpenshiftVersion() (string, error) {
	openShiftCfg := u.client.OpenshiftConfig()
	if openShiftCfg == nil {
		return "", errors.New("failed to get OpenShift config")
	}
	configV1 := openShiftCfg.ConfigV1()
	if configV1 == nil {
		return "", errors.Errorf("invalid OpenShift config, %v", openShiftCfg)
	}
	operators := configV1.ClusterOperators()
	if operators == nil {
		return "", errors.Errorf("cannot get cluster operators from ConfigV1 %v", configV1)
	}
	var clusterOperator *v1.ClusterOperator
	err := retry.WithRetry(
		func() error {
			ctx, cancel := context.WithTimeout(context.Background(), getVersionTimeout)
			defer cancel()
			var err error
			clusterOperator, err = operators.Get(ctx, "openshift-apiserver", metav1.GetOptions{})
			if err != nil {
				if kerrors.IsTimeout(err) || kerrors.IsServerTimeout(err) || kerrors.IsTooManyRequests(err) || kerrors.IsServiceUnavailable(err) {
					return retry.MakeRetryable(err)
				}
				return err
			}
			return nil
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(getVersionRetries),
		retry.OnFailedAttempts(func(err error) {
			log.Errorf("Failed to fetch version %v, retrying in %v", err, getVersionRetryDelay)
			time.Sleep(getVersionRetryDelay)
		}))

	if err != nil {
		return "", err
	}

	for _, ver := range clusterOperator.Status.Versions {
		if ver.Name == "operator" {
			return ver.Version, nil
		}
	}
	return "", nil
}

func (u *updaterImpl) getOpenshiftVersionLegacyAPI() (string, error) {
	var oVersionBody []byte
	err := retry.WithRetry(
		func() error {
			ctx, cancel := context.WithTimeout(context.Background(), getVersionTimeout)
			defer cancel()
			var err error
			oVersionBody, err = u.kubeClient.Discovery().RESTClient().Get().AbsPath("/version/openshift").Do(ctx).Raw()
			if err != nil {
				if kerrors.IsTimeout(err) || kerrors.IsServerTimeout(err) || kerrors.IsTooManyRequests(err) || kerrors.IsServiceUnavailable(err) {
					return retry.MakeRetryable(err)
				}
				return err
			}
			return nil
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(getVersionRetries),
		retry.OnFailedAttempts(func(err error) {
			log.Errorf("Failed to fetch version %v, retrying in %v", err, getVersionRetryDelay)
			time.Sleep(getVersionRetryDelay)
		}))

	if err != nil {
		return "", err
	}
	var ocServerInfo apimachineryversion.Info
	err = json.Unmarshal(oVersionBody, &ocServerInfo)
	if err != nil && len(oVersionBody) > 0 {
		return "", err
	}
	return ocServerInfo.String(), nil
}

// API versions exists as the fields in the kube client.
func (u *updaterImpl) getAPIVersions() []string {
	groupList, err := u.kubeClient.Discovery().ServerGroups()
	if err != nil {
		log.Errorf("unable to fetch api-versions: %s", err)
		return nil
	}

	apiVersions := metav1.ExtractGroupVersions(groupList)
	sort.Strings(apiVersions)
	return apiVersions
}

func (u *updaterImpl) getCloudProviderMetadata(ctx context.Context) *storage.ProviderMetadata {
	m := u.getProviders(ctx)
	if m == nil {
		log.Info("No Cloud Provider metadata is found")
	}
	return m
}

// NewUpdater returns a new ready-to-use updater.
func NewUpdater(client client.Interface) common.SensorComponent {
	offlineMode := &atomic.Bool{}
	offlineMode.Store(true)
	return &updaterImpl{
		client:       client,
		kubeClient:   client.Kubernetes(),
		updates:      make(chan *message.ExpiringMessage),
		stopSig:      concurrency.NewSignal(),
		offlineMode:  offlineMode,
		getProviders: providers.GetMetadata,
	}
}
