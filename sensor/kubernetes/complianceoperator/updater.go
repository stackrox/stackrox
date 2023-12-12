package complianceoperator

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	appsv1 "k8s.io/api/apps/v1"
	kubeAPIErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultInterval = 15 * time.Second
)

var (
	log = logging.LoggerForModule()
)

// NewInfoUpdater return a sensor component that periodically collect information about the compliance operator.
func NewInfoUpdater(client kubernetes.Interface, updateInterval time.Duration) InfoUpdater {
	if updateInterval == 0 {
		updateInterval = defaultInterval
	}
	return &updaterImpl{
		client:         client,
		updateInterval: updateInterval,
		response:       make(chan *message.ExpiringMessage),
		stopSig:        concurrency.NewSignal(),
	}
}

type updaterImpl struct {
	client               kubernetes.Interface
	updateInterval       time.Duration
	response             chan *message.ExpiringMessage
	stopSig              concurrency.Signal
	complianceOperatorNS string
}

func (u *updaterImpl) Start() error {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	go u.run()
	return nil
}

func (u *updaterImpl) Stop(_ error) {
	u.stopSig.Signal()
}

func (u *updaterImpl) Notify(common.SensorComponentEvent) {}

func (u *updaterImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.HealthMonitoringCap}
}

func (u *updaterImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (u *updaterImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return u.response
}

func (u *updaterImpl) GetNamespace() string {
	return u.complianceOperatorNS
}

func (u *updaterImpl) run() {
	ticker := time.NewTicker(u.updateInterval)
	defer ticker.Stop()

	if responseSent := u.collectInfoAndSendResponse(); !responseSent {
		return
	}

	for {
		select {
		case <-ticker.C:
			if responseSent := u.collectInfoAndSendResponse(); !responseSent {
				return
			}
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *updaterImpl) collectInfoAndSendResponse() bool {
	info := u.getComplianceOperatorInfo()
	u.complianceOperatorNS = info.GetNamespace()

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ComplianceOperatorInfo{
			ComplianceOperatorInfo: info,
		},
	}

	log.Debugf("Compliance Operator Info: %v", protoutils.NewWrapper(msg.GetComplianceOperatorInfo()))

	select {
	case u.response <- message.New(msg):
		return true
	case <-u.stopSig.Done():
		return false
	}
}

func (u *updaterImpl) getComplianceOperatorInfo() *central.ComplianceOperatorInfo {
	var err error
	ns := u.complianceOperatorNS
	if ns == "" {
		ns, err = u.getComplianceOperatorNamespace()
		if err != nil {
			return &central.ComplianceOperatorInfo{
				StatusError: err.Error(),
			}
		}
	}

	complianceOperator, err := getComplianceOperator(u.ctx(), u.client, ns)
	if err != nil {
		// Lookup all namespaces again to cover the case that compliance operator was moved to different complianceOperatorNS.
		if kubeAPIErr.IsNotFound(err) {
			ns, err = u.getComplianceOperatorNamespace()
			if err == nil {
				complianceOperator, err = getComplianceOperator(u.ctx(), u.client, ns)
			}
		}
	}
	if err != nil {
		return &central.ComplianceOperatorInfo{
			StatusError: err.Error(),
		}
	}

	var version string
	for key, val := range complianceOperator.Labels {
		if strings.HasSuffix(key, "owner") {
			version = strings.TrimPrefix(val, complianceoperator.Name+".")
		}
	}

	info := &central.ComplianceOperatorInfo{
		Namespace: complianceOperator.GetNamespace(),
		TotalDesiredPodsOpt: &central.ComplianceOperatorInfo_TotalDesiredPods{
			TotalDesiredPods: complianceOperator.Status.Replicas,
		},
		TotalReadyPodsOpt: &central.ComplianceOperatorInfo_TotalReadyPods{
			TotalReadyPods: complianceOperator.Status.ReadyReplicas,
		},
		Version: version,
	}

	resourceList, err := getResourceListForComplianceGroupVersion(u.client)
	if err != nil {
		info.StatusError = err.Error()
		return info
	}

	if err := checkRequiredComplianceCRDsExist(resourceList); err != nil {
		info.StatusError = err.Error()
	}

	return info
}

func (u *updaterImpl) getComplianceOperatorNamespace() (string, error) {
	// List all namespaces to begin the lookup for compliance operator.
	namespaceList, err := u.client.CoreV1().Namespaces().List(u.ctx(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, namespace := range namespaceList.Items {
		complianceOperator, err := getComplianceOperator(u.ctx(), u.client, namespace.Name)
		if err == nil {
			return complianceOperator.GetNamespace(), nil
		}
		// Until we check all namespaces, we cannot determine if compliance operator is installed or not.
		if kubeAPIErr.IsNotFound(err) {
			continue
		}
		return "", err
	}

	return "", errors.Errorf("deployment %s not found in any namespace", complianceoperator.Name)
}

func (u *updaterImpl) ctx() context.Context {
	return concurrency.AsContext(&u.stopSig)
}

func getResourceListForComplianceGroupVersion(client kubernetes.Interface) (*metav1.APIResourceList, error) {
	resourceList, err := client.Discovery().ServerResourcesForGroupVersion(complianceoperator.GetGroupVersion().String())
	if err != nil {
		return nil, err
	}
	if resourceList == nil {
		return nil, errors.Errorf("API group-version %q not found", complianceoperator.GetGroupVersion().String())
	}
	return resourceList, nil
}

func checkRequiredComplianceCRDsExist(resourceList *metav1.APIResourceList) error {
	if resourceList == nil {
		return errors.New("could not determine required GroupVersionKinds")
	}

	detectedKinds := set.NewStringSet()
	for _, resource := range resourceList.APIResources {
		detectedKinds.Add(resource.Kind)
	}

	errorList := errorhelpers.NewErrorList("checking for CRDs required for compliance")
	for _, requiredResource := range complianceoperator.GetRequiredResources() {
		if !detectedKinds.Contains(requiredResource.Kind) {
			errorList.AddError(errors.Errorf("required GroupVersionKind %q not found", requiredResource.GroupVersionKind().String()))
		}
	}
	return errorList.ToError()
}

func getComplianceOperator(ctx context.Context, client kubernetes.Interface, namespace string) (*appsv1.Deployment, error) {
	return client.AppsV1().Deployments(namespace).Get(ctx, complianceoperator.Name, metav1.GetOptions{})
}
