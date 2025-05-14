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
	"github.com/stackrox/rox/pkg/k8sintrospect"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/telemetry"
	v1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultInterval = 15 * time.Second
)

var (
	log  = logging.LoggerForModule()
	once = sync.Once{}
)

// registerDiagnosticComplianceOperatorObjects adds compliance operator objects to the default objects to pull in the diagnostic bundles
func (u *updaterImpl) registerDiagnosticComplianceOperatorObjects(info *central.ComplianceOperatorInfo) {
	registerFunc := func(req *central.PullTelemetryDataRequest, cfg k8sintrospect.Config) k8sintrospect.Config {
		if !req.GetWithComplianceOperator() {
			log.Info("Skipping adding compliance operator objects to diagnostic bundles")
			return cfg
		}

		cfg.Objects = append(cfg.Objects, k8sintrospect.ObjectConfig{
			GVK: complianceoperator.ComplianceScan.GroupVersionKind(),
		}, k8sintrospect.ObjectConfig{
			GVK: complianceoperator.ComplianceSuite.GroupVersionKind(),
		}, k8sintrospect.ObjectConfig{
			GVK: complianceoperator.ComplianceRemediation.GroupVersionKind(),
		}, k8sintrospect.ObjectConfig{
			GVK: complianceoperator.ComplianceCheckResult.GroupVersionKind(),
		}, k8sintrospect.ObjectConfig{
			GVK: complianceoperator.Profile.GroupVersionKind(),
		}, k8sintrospect.ObjectConfig{
			GVK: complianceoperator.TailoredProfile.GroupVersionKind(),
		}, k8sintrospect.ObjectConfig{
			GVK: complianceoperator.ScanSettingBinding.GroupVersionKind(),
		}, k8sintrospect.ObjectConfig{
			GVK: complianceoperator.ScanSetting.GroupVersionKind(),
		}, k8sintrospect.ObjectConfig{
			GVK: complianceoperator.Rule.GroupVersionKind(),
		})
		cfg.Namespaces = append(cfg.Namespaces, info.GetNamespace())
		return cfg
	}

	once.Do(func() {
		telemetry.RegisterDiagnosticConfigurationFunc(registerFunc)
	})
}

// NewInfoUpdater return a sensor component that periodically collect information about the compliance operator.
func NewInfoUpdater(client kubernetes.Interface, updateInterval time.Duration, readySignal *concurrency.Signal) InfoUpdater {
	if updateInterval == 0 {
		updateInterval = defaultInterval
	}
	updateTicker := time.NewTicker(updateInterval)
	updateTicker.Stop()
	// We start the signal untriggered
	readySignal.Reset()
	return &updaterImpl{
		client:               client,
		updateInterval:       updateInterval,
		response:             make(chan *message.ExpiringMessage),
		stopSig:              concurrency.NewSignal(),
		updateTicker:         updateTicker,
		complianceOperatorNS: "openshift-compliance",
		isReady:              readySignal,
	}
}

type updaterImpl struct {
	client               kubernetes.Interface
	updateTicker         *time.Ticker
	updateInterval       time.Duration
	response             chan *message.ExpiringMessage
	stopSig              concurrency.Signal
	complianceOperatorNS string
	isReady              *concurrency.Signal
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
	case common.SensorComponentEventOfflineMode:
		u.isReady.Reset()
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
	if info.GetIsInstalled() {
		u.isReady.Signal()
	} else {
		u.isReady.Reset()
	}

	// Register compliance operator objects for diagnostic bundles if it was found.
	if info.GetIsInstalled() && info.GetNamespace() != "" {
		u.registerDiagnosticComplianceOperatorObjects(info)
	}

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
	complianceOperatorDeployment, err := searchForDeployment(u.complianceOperatorNS, u.client, u.ctx())
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
