package clusterhealth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
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
)

var (
	log = logging.LoggerForModule()
)

type updaterImpl struct {
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

func (u *updaterImpl) Start() error {
	go u.run(u.updateTicker.C)
	return nil
}

func (u *updaterImpl) Stop(_ error) {
	u.updateTicker.Stop()
	u.stopSig.Signal()
}

func (u *updaterImpl) Notify(e common.SensorComponentEvent) {
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

func (u *updaterImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
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
	var result storage.ScannerHealthInfo
	// Local Scanner deployment is looked up in the same namespace as Sensor because that is how they should be deployed.
	localScanner, err := u.client.AppsV1().Deployments(u.namespace).Get(u.ctx(), localScannerDeploymentName, metav1.GetOptions{})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("unable to find local scanner deployment in namespace %q", u.namespace))
		result.StatusErrors = append(result.StatusErrors, fmt.Sprintf("unable to find local scanner deployment in namespace %q: %v", u.namespace, err))
	} else {
		result.TotalDesiredAnalyzerPodsOpt = &storage.ScannerHealthInfo_TotalDesiredAnalyzerPods{
			TotalDesiredAnalyzerPods: localScanner.Status.Replicas,
		}
		result.TotalReadyAnalyzerPodsOpt = &storage.ScannerHealthInfo_TotalReadyAnalyzerPods{
			TotalReadyAnalyzerPods: localScanner.Status.ReadyReplicas,
		}
	}
	localScannerDB, err := u.client.AppsV1().Deployments(u.namespace).Get(u.ctx(), localScannerDBDeploymentName, metav1.GetOptions{})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("unable to find local scanner DB deployment in namespace %q", u.namespace))
		result.StatusErrors = append(result.StatusErrors, fmt.Sprintf("unable to find local scanner DB deployment in namespace %q: %v", u.namespace, err))
	} else {
		result.TotalDesiredDbPodsOpt = &storage.ScannerHealthInfo_TotalDesiredDbPods{
			TotalDesiredDbPods: localScannerDB.Status.Replicas,
		}
		result.TotalReadyDbPodsOpt = &storage.ScannerHealthInfo_TotalReadyDbPods{
			TotalReadyDbPods: localScannerDB.Status.ReadyReplicas,
		}
	}

	// TODO/MC: Extend for ScannerV4 case?

	if len(result.StatusErrors) > 0 {
		log.Errorf("Errors while getting local scanner info: %v", result.StatusErrors)
	}

	return &result
}

func getSensorNamespace() string {
	// The corresponding environment variable is configured to contain pod namespace by sensor YAML/helm file.
	const nsEnvVar = "POD_NAMESPACE"
	ns := os.Getenv(nsEnvVar)
	if ns == "" {
		ns = namespaces.StackRox
		log.Warnf("%s environment variable is unset/empty, using %q as fallback for sensor namespace", nsEnvVar, ns)
	}
	return ns
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
		namespace:      getSensorNamespace(),
		updateTicker:   updateTicker,
	}
}
