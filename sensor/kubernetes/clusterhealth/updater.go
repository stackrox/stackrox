package clusterhealth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pods"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/unimplemented"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

const (
	defaultInterval        = 30 * time.Second
	collectorDaemonsetName = "collector"
	collectorContainerName = "collector"

	admissionControlDeploymentName = "admission-control"

	localScannerDeploymentName   = "scanner"
	localScannerDBDeploymentName = "scanner-db"

	localScannerV4IndexerDeploymentName = "scanner-v4-indexer"
	localScannerV4DBDeploymentName      = "scanner-v4-db"
)

var (
	log = logging.LoggerForModule()
)

type updaterImpl struct {
	unimplemented.Receiver

	dynClient      dynamic.Interface
	updates        chan *message.ExpiringMessage
	stopSig        concurrency.Signal
	updateInterval time.Duration
	namespace      string
	updateTicker   *time.Ticker
	ctxMutex       sync.Mutex
	pipelineCtx    context.Context
	cancelCtx      context.CancelFunc
}

func (u *updaterImpl) Name() string {
	return "clusterhealth.updaterImpl"
}

func (u *updaterImpl) Start() error {
	go u.run(u.updateTicker.C)
	return nil
}

func (u *updaterImpl) Stop() {
	u.updateTicker.Stop()
	u.stopSig.Signal()
}

func (u *updaterImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		u.resetContext()
		u.updateTicker.Reset(u.updateInterval)
	case common.SensorComponentEventOfflineMode:
		u.updateTicker.Stop()
	}
}

func (u *updaterImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.HealthMonitoringCap}
}

func (u *updaterImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return u.updates
}

func (u *updaterImpl) resetContext() {
	u.ctxMutex.Lock()
	defer u.ctxMutex.Unlock()
	if u.cancelCtx != nil {
		u.cancelCtx()
	}
	u.pipelineCtx, u.cancelCtx = context.WithCancel(context.Background())
}

func (u *updaterImpl) getCurrentContext() context.Context {
	u.ctxMutex.Lock()
	defer u.ctxMutex.Unlock()
	return u.pipelineCtx
}

func (u *updaterImpl) run(tickerC <-chan time.Time) {
	const refreshEveryN = 10
	tickCount := 0
	var cachedHealth *central.RawClusterHealthInfo
	for {
		select {
		case <-tickerC:
			if cachedHealth == nil || tickCount%refreshEveryN == 0 {
				cachedHealth = &central.RawClusterHealthInfo{
					CollectorHealthInfo:        u.getCollectorInfo(),
					AdmissionControlHealthInfo: u.getAdmissionControlInfo(),
					ScannerHealthInfo:          u.getLocalScannerInfo(),
				}
			}
			tickCount++
			select {
			case u.updates <- message.NewExpiring(u.getCurrentContext(), &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_ClusterHealthInfo{
					ClusterHealthInfo: cachedHealth,
				},
			}):
				continue
			case <-u.stopSig.Done():
				return
			}
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *updaterImpl) getCollectorInfo() *storage.CollectorHealthInfo {
	result := storage.CollectorHealthInfo{}

	nodeList, err := u.dynClient.Resource(client.NodeGVR).List(u.ctx(), metav1.ListOptions{})
	if err != nil {
		result.StatusErrors = append(result.StatusErrors, errors.Wrap(err, "unable to list cluster nodes").Error())
	} else {
		result.TotalRegisteredNodesOpt = &storage.CollectorHealthInfo_TotalRegisteredNodes{
			TotalRegisteredNodes: int32(len(nodeList.Items)),
		}
	}

	unstructuredDS, err := u.dynClient.Resource(client.DaemonSetGVR).Namespace(u.namespace).Get(u.ctx(), collectorDaemonsetName, metav1.GetOptions{})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("unable to find collector DaemonSet in namespace %q", u.namespace))
		result.StatusErrors = append(result.StatusErrors, err.Error())
	} else {
		var collectorDS appsv1.DaemonSet
		if convErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredDS.Object, &collectorDS); convErr != nil {
			result.StatusErrors = append(result.StatusErrors, convErr.Error())
		} else {
			for _, container := range collectorDS.Spec.Template.Spec.Containers {
				if container.Name == collectorContainerName {
					result.Version = stringutils.GetAfterLast(container.Image, ":")
					result.Version = strings.TrimSuffix(result.GetVersion(), "-slim")
					result.Version = strings.TrimSuffix(result.GetVersion(), "-latest")
					break
				}
			}
			if result.GetVersion() == "" {
				result.StatusErrors = append(result.StatusErrors, "unable to determine collector version")
			}

			result.TotalDesiredPodsOpt = &storage.CollectorHealthInfo_TotalDesiredPods{
				TotalDesiredPods: collectorDS.Status.DesiredNumberScheduled,
			}
			result.TotalReadyPodsOpt = &storage.CollectorHealthInfo_TotalReadyPods{
				TotalReadyPods: collectorDS.Status.NumberReady,
			}
		}
	}

	if len(result.GetStatusErrors()) > 0 {
		log.Errorf("Errors while getting collector info: %v", result.GetStatusErrors())
	}

	return &result
}

func (u *updaterImpl) getDeploymentStatus(name string) (*appsv1.DeploymentStatus, error) {
	unstructuredDeploy, err := u.dynClient.Resource(client.DeploymentGVR).Namespace(u.namespace).Get(u.ctx(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var deploy appsv1.Deployment
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredDeploy.Object, &deploy); err != nil {
		return nil, err
	}
	return &deploy.Status, nil
}

func (u *updaterImpl) getAdmissionControlInfo() *storage.AdmissionControlHealthInfo {
	result := storage.AdmissionControlHealthInfo{}
	status, err := u.getDeploymentStatus(admissionControlDeploymentName)
	if err != nil {
		result.StatusErrors = append(result.StatusErrors, fmt.Sprintf("unable to find admission control deployments in namespace %q: %v", u.namespace, err))
	} else {
		result.TotalDesiredPodsOpt = &storage.AdmissionControlHealthInfo_TotalDesiredPods{
			TotalDesiredPods: status.Replicas,
		}
		result.TotalReadyPodsOpt = &storage.AdmissionControlHealthInfo_TotalReadyPods{
			TotalReadyPods: status.ReadyReplicas,
		}
	}

	if len(result.GetStatusErrors()) > 0 {
		log.Errorf("Errors while getting admission control info: %v", result.GetStatusErrors())
	}
	return &result
}

func (u *updaterImpl) getLocalScannerInfo() *storage.ScannerHealthInfo {
	if !env.LocalImageScanningEnabled.BooleanSetting() {
		return nil
	}

	scannerV4Active := features.ScannerV4.Enabled() && centralcaps.Has(centralsensor.ScannerV4Supported)

	analyzerDeploymentName := localScannerDeploymentName
	dbDeploymentName := localScannerDBDeploymentName
	if scannerV4Active {
		analyzerDeploymentName = localScannerV4IndexerDeploymentName
		dbDeploymentName = localScannerV4DBDeploymentName
	}

	result := u.getScannerHealthInfo(analyzerDeploymentName, dbDeploymentName)
	if len(result.GetStatusErrors()) > 0 {
		log.Errorf("Errors while getting local scanner info: %v", result.GetStatusErrors())
	}

	return result
}

func (u *updaterImpl) getScannerHealthInfo(analyzerDeployName string, dbDeployName string) *storage.ScannerHealthInfo {
	var result storage.ScannerHealthInfo

	analyzerStatus, err := u.getDeploymentStatus(analyzerDeployName)
	if err != nil {
		result.StatusErrors = append(result.StatusErrors, fmt.Sprintf("unable to find %q deployment in namespace %q: %v", analyzerDeployName, u.namespace, err))
	} else {
		result.TotalDesiredAnalyzerPodsOpt = &storage.ScannerHealthInfo_TotalDesiredAnalyzerPods{
			TotalDesiredAnalyzerPods: analyzerStatus.Replicas,
		}
		result.TotalReadyAnalyzerPodsOpt = &storage.ScannerHealthInfo_TotalReadyAnalyzerPods{
			TotalReadyAnalyzerPods: analyzerStatus.ReadyReplicas,
		}
	}
	dbStatus, err := u.getDeploymentStatus(dbDeployName)
	if err != nil {
		result.StatusErrors = append(result.StatusErrors, fmt.Sprintf("unable to find %q deployment in namespace %q: %v", dbDeployName, u.namespace, err))
	} else {
		result.TotalDesiredDbPodsOpt = &storage.ScannerHealthInfo_TotalDesiredDbPods{
			TotalDesiredDbPods: dbStatus.Replicas,
		}
		result.TotalReadyDbPodsOpt = &storage.ScannerHealthInfo_TotalReadyDbPods{
			TotalReadyDbPods: dbStatus.ReadyReplicas,
		}
	}

	return &result
}

func (u *updaterImpl) ctx() context.Context {
	return concurrency.AsContext(&u.stopSig)
}

// NewUpdater returns a new ready-to-use updater.
// updateInterval is optional argument, default 30 seconds interval is used.
func NewUpdater(dynClient dynamic.Interface, updateInterval time.Duration) common.SensorComponent {
	interval := updateInterval
	if interval == 0 {
		interval = defaultInterval
	}
	updateTicker := time.NewTicker(interval)
	updateTicker.Stop()
	return &updaterImpl{
		dynClient:      dynClient,
		updates:        make(chan *message.ExpiringMessage),
		stopSig:        concurrency.NewSignal(),
		updateInterval: interval,
		namespace:      pods.GetPodNamespace(),
		updateTicker:   updateTicker,
	}
}
