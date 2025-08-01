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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultInterval        = 30 * time.Second
	collectorDaemonsetName = "collector"
	collectorContainerName = "collector"

	admissionControlDeploymentName = "admission-control"
	admissionControlContainerName  = "admission-control"

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

	client         kubernetes.Interface
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
	for {
		select {
		case <-tickerC:
			collectorHealthInfo := u.getCollectorInfo()
			admissionControlHealthInfo := u.getAdmissionControlInfo()
			scannerHealthInfo := u.getLocalScannerInfo()
			select {
			case u.updates <- message.NewExpiring(u.getCurrentContext(), &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_ClusterHealthInfo{
					ClusterHealthInfo: &central.RawClusterHealthInfo{
						CollectorHealthInfo:        collectorHealthInfo,
						AdmissionControlHealthInfo: admissionControlHealthInfo,
						ScannerHealthInfo:          scannerHealthInfo,
					},
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

	nodes, err := u.client.CoreV1().Nodes().List(u.ctx(), metav1.ListOptions{})
	if err != nil {
		result.StatusErrors = append(result.StatusErrors, errors.Wrap(err, "unable to list cluster nodes").Error())
	} else {
		result.TotalRegisteredNodesOpt = &storage.CollectorHealthInfo_TotalRegisteredNodes{
			TotalRegisteredNodes: int32(len(nodes.Items)),
		}
	}

	// Collector DaemonSet is looked up in the same namespace as Sensor because that is how they should be deployed.
	collectorDS, err := u.client.AppsV1().DaemonSets(u.namespace).Get(u.ctx(), collectorDaemonsetName, metav1.GetOptions{})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("unable to find collector DaemonSet in namespace %q", u.namespace))
		result.StatusErrors = append(result.StatusErrors, err.Error())
	} else {
		for _, container := range collectorDS.Spec.Template.Spec.Containers {
			if container.Name == collectorContainerName {
				result.Version = stringutils.GetAfterLast(container.Image, ":")
				result.Version = strings.TrimSuffix(result.Version, "-slim")
				result.Version = strings.TrimSuffix(result.Version, "-latest")
				break
			}
		}
		if result.Version == "" {
			result.StatusErrors = append(result.StatusErrors, "unable to determine collector version")
		}

		result.TotalDesiredPodsOpt = &storage.CollectorHealthInfo_TotalDesiredPods{
			TotalDesiredPods: collectorDS.Status.DesiredNumberScheduled,
		}
		result.TotalReadyPodsOpt = &storage.CollectorHealthInfo_TotalReadyPods{
			TotalReadyPods: collectorDS.Status.NumberReady,
		}
	}

	if len(result.StatusErrors) > 0 {
		log.Errorf("Errors while getting collector info: %v", result.StatusErrors)
	}

	return &result
}

func (u *updaterImpl) getAdmissionControlInfo() *storage.AdmissionControlHealthInfo {
	result := storage.AdmissionControlHealthInfo{}
	// Admission Control deployment is looked up in the same namespace as Sensor because that is how they should be deployed.
	admissionControl, err := u.client.AppsV1().Deployments(u.namespace).Get(u.ctx(), admissionControlDeploymentName, metav1.GetOptions{})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("unable to find admission control deployments in namespace %q", u.namespace))
		result.StatusErrors = append(result.StatusErrors, fmt.Sprintf("unable to find admission control deployments in namespace %q: %v", u.namespace, err))
	} else {
		result.TotalDesiredPodsOpt = &storage.AdmissionControlHealthInfo_TotalDesiredPods{
			TotalDesiredPods: admissionControl.Status.Replicas,
		}
		result.TotalReadyPodsOpt = &storage.AdmissionControlHealthInfo_TotalReadyPods{
			TotalReadyPods: admissionControl.Status.ReadyReplicas,
		}
	}

	if len(result.StatusErrors) > 0 {
		log.Errorf("Errors while getting admission control info: %v", result.StatusErrors)
	}
	return &result
}

func (u *updaterImpl) getLocalScannerInfo() *storage.ScannerHealthInfo {
	if !env.LocalImageScanningEnabled.BooleanSetting() {
		return nil
	}

	// It's possible that both Scanner and Scanner V4 are installed in the secured cluster
	// at the same time, but only one will be used by Sensor at any given time, therefore
	// only report the health of the active scanner.
	scannerV4Active := features.ScannerV4.Enabled() && centralcaps.Has(centralsensor.ScannerV4Supported)

	analyzerDeploymentName := localScannerDeploymentName
	dbDeploymentName := localScannerDBDeploymentName
	if scannerV4Active {
		analyzerDeploymentName = localScannerV4IndexerDeploymentName
		dbDeploymentName = localScannerV4DBDeploymentName
	}

	result := u.getScannerHealthInfo(analyzerDeploymentName, dbDeploymentName)
	if len(result.StatusErrors) > 0 {
		log.Errorf("Errors while getting local scanner info: %v", result.StatusErrors)
	}

	return result
}

func (u *updaterImpl) getScannerHealthInfo(analyzerDeployName string, dbDeployName string) *storage.ScannerHealthInfo {
	var result storage.ScannerHealthInfo

	// Local Scanner deployment is looked up in the same namespace as Sensor because that is how they should be deployed.
	localScanner, err := u.client.AppsV1().Deployments(u.namespace).Get(u.ctx(), analyzerDeployName, metav1.GetOptions{})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("unable to find %q deployment in namespace %q", analyzerDeployName, u.namespace))
		result.StatusErrors = append(result.StatusErrors, err.Error())
	} else {
		result.TotalDesiredAnalyzerPodsOpt = &storage.ScannerHealthInfo_TotalDesiredAnalyzerPods{
			TotalDesiredAnalyzerPods: localScanner.Status.Replicas,
		}
		result.TotalReadyAnalyzerPodsOpt = &storage.ScannerHealthInfo_TotalReadyAnalyzerPods{
			TotalReadyAnalyzerPods: localScanner.Status.ReadyReplicas,
		}
	}
	localScannerDB, err := u.client.AppsV1().Deployments(u.namespace).Get(u.ctx(), dbDeployName, metav1.GetOptions{})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("unable to find %q deployment in namespace %q", dbDeployName, u.namespace))
		result.StatusErrors = append(result.StatusErrors, err.Error())
	} else {
		result.TotalDesiredDbPodsOpt = &storage.ScannerHealthInfo_TotalDesiredDbPods{
			TotalDesiredDbPods: localScannerDB.Status.Replicas,
		}
		result.TotalReadyDbPodsOpt = &storage.ScannerHealthInfo_TotalReadyDbPods{
			TotalReadyDbPods: localScannerDB.Status.ReadyReplicas,
		}
	}

	return &result
}

func (u *updaterImpl) ctx() context.Context {
	return concurrency.AsContext(&u.stopSig)
}

// NewUpdater returns a new ready-to-use updater.
// updateInterval is optional argument, default 30 seconds interval is used.
func NewUpdater(client kubernetes.Interface, updateInterval time.Duration) common.SensorComponent {
	interval := updateInterval
	if interval == 0 {
		interval = defaultInterval
	}
	updateTicker := time.NewTicker(interval)
	updateTicker.Stop()
	return &updaterImpl{
		client:         client,
		updates:        make(chan *message.ExpiringMessage),
		stopSig:        concurrency.NewSignal(),
		updateInterval: interval,
		namespace:      pods.GetPodNamespace(),
		updateTicker:   updateTicker,
	}
}
