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
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/authorization/v1"
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
	updateTicker := time.NewTicker(updateInterval)
	updateTicker.Stop()
	return &updaterImpl{
		client:               client,
		updateInterval:       updateInterval,
		response:             make(chan *message.ExpiringMessage),
		stopSig:              concurrency.NewSignal(),
		updateTicker:         updateTicker,
		complianceOperatorNS: "openshift-compliance",
	}
}

type updaterImpl struct {
	client               kubernetes.Interface
	updateTicker         *time.Ticker
	updateInterval       time.Duration
	response             chan *message.ExpiringMessage
	stopSig              concurrency.Signal
	complianceOperatorNS string
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
	case common.SensorComponentEventSyncFinished:
		if centralcaps.Has(centralsensor.ComplianceV2Integrations) {
			u.updateTicker.Reset(u.updateInterval)
			return
		}
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
	return u.response
}

func (u *updaterImpl) GetNamespace() string {
	return u.complianceOperatorNS
}

func (u *updaterImpl) run(tickerC <-chan time.Time) {
	if responseSent := u.collectInfoAndSendResponse(); !responseSent {
		return
	}

	for {
		select {
		case <-tickerC:
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
	complianceOperatorDeployment, err := u.searchComplianceOperatorDeployment()
	if err != nil {
		return &central.ComplianceOperatorInfo{
			StatusError: err.Error(),
			IsInstalled: false,
		}
	}

	var version string
	for key, val := range complianceOperatorDeployment.Labels {
		// Info: This label is set by OLM, if a custom compliance operator build was deployed via e.g. Helm, this label does not exist.
		if strings.HasSuffix(key, "owner") {
			version = strings.TrimPrefix(val, complianceoperator.Name+".")
		}
	}

	info := &central.ComplianceOperatorInfo{
		Namespace: complianceOperatorDeployment.GetNamespace(),
		TotalDesiredPodsOpt: &central.ComplianceOperatorInfo_TotalDesiredPods{
			TotalDesiredPods: complianceOperatorDeployment.Status.Replicas,
		},
		TotalReadyPodsOpt: &central.ComplianceOperatorInfo_TotalReadyPods{
			TotalReadyPods: complianceOperatorDeployment.Status.ReadyReplicas,
		},
		Version:     version,
		IsInstalled: true,
	}

	// Check Sensor access to compliance.openshift.io resources
	if err := checkWriteAccess(u.client); err != nil {
		info.StatusError = err.Error()
		return info
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

// checkWriteAccess checks if Sensor has permissions to write to compliance operator CRs.
func checkWriteAccess(client kubernetes.Interface) error {
	sac := &v1.SelfSubjectAccessReview{
		Spec: v1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &v1.ResourceAttributes{
				Verb:     "*",
				Resource: "*",
				Group:    "compliance.openshift.io",
			},
		},
	}

	response, err := client.AuthorizationV1().SelfSubjectAccessReviews().Create(context.Background(), sac, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "could not perform compliance operator access review")
	}

	if !response.Status.Allowed {
		return errors.New("Sensor cannot write compliance.openshift.io API group resources. Please check Sensor's RBAC permissions.")
	}
	return nil
}

func (u *updaterImpl) searchComplianceOperatorDeployment() (*appsv1.Deployment, error) {
	// Use cached namespace, if compliance operator deployment was not found search again in all namespaces.
	if u.complianceOperatorNS != "" {
		if complianceOperator, err := u.getComplianceOperatorDeployment(u.complianceOperatorNS); err == nil {
			return complianceOperator, nil
		}
	}

	// List all namespaces to begin the lookup for compliance operator.
	namespaceList, err := u.client.CoreV1().Namespaces().List(u.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, namespace := range namespaceList.Items {
		complianceOperator, err := u.getComplianceOperatorDeployment(namespace.GetName())
		if err == nil {
			return complianceOperator, nil
		}
		// Until we check all namespaces, we cannot determine if compliance operator is installed or not.
		if kubeAPIErr.IsNotFound(err) {
			continue
		}
		return nil, err
	}

	return nil, errors.Errorf("deployment %s not found in any namespace", complianceoperator.Name)
}

func (u *updaterImpl) getComplianceOperatorDeployment(ns string) (*appsv1.Deployment, error) {
	return u.client.AppsV1().Deployments(ns).Get(u.ctx(), complianceoperator.Name, metav1.GetOptions{})
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
