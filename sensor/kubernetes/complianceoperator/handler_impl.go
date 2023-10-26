package complianceoperator

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	kubeAPIErr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/pointer"
)

type handlerImpl struct {
	client                 dynamic.Interface
	complianceOperatorInfo StatusInfo

	response chan *message.ExpiringMessage
	request  chan *central.ComplianceRequest

	disabled   concurrency.Signal
	stopSignal concurrency.Signal
}

type ScanScheduleConfiguration struct {
	Suspend        *bool
	Schedule       *string
	ScanName       string
	Request        interface{}
	ValidationFunc func(interface{}) error
}

// NewRequestHandler returns instance of common.SensorComponent interface which can handle compliance requests from Central.
func NewRequestHandler(client dynamic.Interface, complianceOperatorInfo StatusInfo) common.SensorComponent {
	return &handlerImpl{
		client:                 client,
		complianceOperatorInfo: complianceOperatorInfo,

		request:  make(chan *central.ComplianceRequest),
		response: make(chan *message.ExpiringMessage),

		disabled:   concurrency.NewSignal(),
		stopSignal: concurrency.NewSignal(),
	}
}

func (m *handlerImpl) Start() error {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	// TODO: create default scan setting for ad-hoc scan
	go m.run()
	return nil
}

func (m *handlerImpl) Stop(_ error) {
	m.stopSignal.Signal()
}

func (m *handlerImpl) Notify(common.SensorComponentEvent) {}

func (m *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (m *handlerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	req := msg.GetComplianceRequest()
	if req == nil {
		return nil
	}

	select {
	case m.request <- req:
		return nil
	case <-m.stopSignal.Done():
		return errors.Errorf("Could not process compliance request: %s", protoutils.NewWrapper(msg))
	}
}

func (m *handlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return m.response
}

func (m *handlerImpl) run() {
	for !m.stopSignal.IsDone() {
		select {
		case req := <-m.request:
			var requestProcessed bool
			switch r := req.GetRequest().(type) {
			case *central.ComplianceRequest_EnableCompliance:
				requestProcessed = m.enableCompliance(r.EnableCompliance)
			case *central.ComplianceRequest_DisableCompliance:
				requestProcessed = m.disableCompliance(r.DisableCompliance)
			case *central.ComplianceRequest_ApplyScanConfig:
				requestProcessed = m.processApplyScanCfgRequest(r.ApplyScanConfig)
			case *central.ComplianceRequest_DeleteScanConfig:
				requestProcessed = m.processDeleteScanCfgRequest(r.DeleteScanConfig)
			}

			if !requestProcessed {
				log.Errorf("Could not send response for compliance request: %s", protoutils.NewWrapper(req))
			}
		case <-m.stopSignal.Done():
			return
		}
	}
}

func (m *handlerImpl) enableCompliance(request *central.EnableComplianceRequest) bool {
	m.disabled.Reset()
	// TODO: [ROX-18096] Start collecting compliance profiles & rules
	return m.composeAndSendEnableComplianceResponse(request.GetId(), nil)
}

func (m *handlerImpl) disableCompliance(request *central.DisableComplianceRequest) bool {
	// Disabling compliance should not disable compliance operator status monitoring. Users must be informed about
	// the compliance operator health.

	// TODO: Pause all scans. Currently, the only way a compliance scan can be disabled is by deleting the ScanSettingBinding.
	// TODO: Drop all custom resource listener events.
	m.disabled.Signal()
	return m.composeAndSendDisableComplianceResponse(request.GetId(), nil)
}

func (m *handlerImpl) processApplyScanCfgRequest(request *central.ApplyComplianceScanConfigRequest) bool {
	select {
	case <-m.disabled.Done():
		err := errors.Errorf("Compliance is disabled. Cannot process request: %s", protoutils.NewWrapper(request))
		return m.composeAndSendApplyScanConfigResponse(request.GetId(), err)
	case <-m.stopSignal.Done():
		return true
	default:
		if request.GetScanRequest() == nil {
			return m.composeAndSendApplyScanConfigResponse(request.GetId(), errors.New("Compliance scan request is empty"))
		}

		switch r := request.GetScanRequest().(type) {
		case *central.ApplyComplianceScanConfigRequest_OneTimeScan_:
			return m.processOneTimeScanRequest(request.GetId(), r.OneTimeScan)
		case *central.ApplyComplianceScanConfigRequest_ScheduledScan_:
			return m.processScheduledScanRequest(request.GetId(), r.ScheduledScan)
		case *central.ApplyComplianceScanConfigRequest_SuspendScan:
			return m.processSuspendScheduledScanRequest(request.GetId(), r.SuspendScan)
		case *central.ApplyComplianceScanConfigRequest_ResumeScan:
			return m.processResumeScheduledScanRequest(request.GetId(), r.ResumeScan)
		case *central.ApplyComplianceScanConfigRequest_RerunScan:
			return m.processRerunScheduledScanRequest(request.GetId(), r.RerunScan)
		default:
			return m.composeAndSendApplyScanConfigResponse(request.GetId(), errors.New("Cannot handle compliance scan request"))
		}
	}
}

func (m *handlerImpl) processOneTimeScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_OneTimeScan) bool {
	if err := validateApplyOneTimeScanConfigRequest(request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}
	// TODO: Check if default ACS scan setting CR exists. If it doesn't exist, create one.

	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}

	scanSettingBinding, err := runtimeObjToUnstructured(convertCentralRequestToScanSettingBinding(ns, request.GetScanSettings()))
	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	_, err = m.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns).Create(m.ctx(), scanSettingBinding, v1.CreateOptions{})
	if err != nil {
		err = errors.Wrapf(err, "Could not create namespaces/%s/scansettingbindings/%s", ns, scanSettingBinding.GetName())
	}
	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}

func (m *handlerImpl) processScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_ScheduledScan) bool {
	if err := validateApplyScheduledScanConfigRequest(request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}

	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}

	scanSetting, err := runtimeObjToUnstructured(convertCentralRequestToScanSetting(ns, request))
	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	scanSettingBinding, err := runtimeObjToUnstructured(convertCentralRequestToScanSettingBinding(ns, request.GetScanSettings()))
	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	_, err = m.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns).Create(m.ctx(), scanSetting, v1.CreateOptions{})
	if err != nil {
		err = errors.Wrapf(err, "Could not create namespaces/%s/scansettings/%s", ns, scanSetting.GetName())
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	_, err = m.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns).Create(m.ctx(), scanSettingBinding, v1.CreateOptions{})
	if err != nil {
		err = errors.Wrapf(err, "Could not create namespaces/%s/scansettingbindings/%s", ns, scanSettingBinding.GetName())
	}
	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}

func (m *handlerImpl) processRerunScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_RerunScheduledScan) bool {
	if err := validateApplyRerunScheduledScanRequest(request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}

	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}

	// Conventionally, compliance scan can be rerun by applying an annotation to ComplianceScan CR. Note that the CRs
	// created from a scan configuration have the same name as scan configuration.
	resI := m.client.Resource(complianceoperator.ComplianceScan.GroupVersionResource()).Namespace(ns)
	obj, err := resI.Get(m.ctx(), request.GetScanName(), v1.GetOptions{})
	if err != nil || obj == nil {
		err = errors.Wrapf(err, "namespaces/%s/compliancescans/%s not found", ns, request.GetScanName())
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	var complianceScan v1alpha1.ComplianceScan
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &complianceScan); err != nil {
		err = errors.Wrap(err, "Could not convert unstructured to compliance scan")
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	// Apply annotation to indicate compliance scan be rerun.
	if complianceScan.GetAnnotations() == nil {
		complianceScan.Annotations = make(map[string]string)
	}
	complianceScan.Annotations[rescanAnnotation] = ""

	obj, err = runtimeObjToUnstructured(&complianceScan)
	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}
	_, err = resI.Update(m.ctx(), obj, v1.UpdateOptions{})
	if err != nil {
		err = errors.Wrapf(err, "Could not update namespaces/%s/compliancescans/%s", ns, request.GetScanName())
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}
	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}
func (m *handlerImpl) processScanConfigScheduleChangeRequest(requestID string, config ScanScheduleConfiguration) bool {
	if err := config.ValidationFunc(config.Request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}

	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}

	resI := m.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
	obj, err := resI.Get(m.ctx(), config.ScanName, v1.GetOptions{})
	if err != nil || obj == nil {
		err = errors.Wrapf(err, "namespaces/%s/scansettings/%s not found", ns, config.ScanName)
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	var scanSetting v1alpha1.ScanSetting
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &scanSetting); err != nil {
		err = errors.Wrap(err, "Could not convert unstructured to scan setting")
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	if config.Suspend != nil {
		// TODO: Waiting to get a new release of compliance-operator with
		// the new field added to ScanSetting CRD.
		// Check if scanSetting has the "Suspend" field
		if _, ok := reflect.TypeOf(scanSetting).Elem().FieldByName("Suspend"); ok {
			// scanSetting.Suspend = *config.Suspend // uncomment when ready
			// Field exists, safe to set
		} else {
			// Handle the case where the field doesn't exist (older CRD)
			return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Cannot suspend/resume scheduled scan. Compliance operator version is too old"))
		}
	}

	if config.Schedule != nil {
		scanSetting.Schedule = *config.Schedule
	}

	obj, err = runtimeObjToUnstructured(&scanSetting)
	if err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, err)
	}

	_, err = resI.Update(m.ctx(), obj, v1.UpdateOptions{})
	if err != nil {
		err = errors.Wrapf(err, "Could not update namespaces/%s/scansettings/%s", ns, config.ScanName)
	}

	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}

func validateInterface(i interface{}) error {
	switch req := i.(type) {
	case *central.ApplyComplianceScanConfigRequest_SuspendScheduledScan:
		return validateApplySuspendScheduledScanRequest(req)
	case *central.ApplyComplianceScanConfigRequest_ResumeScheduledScan:
		return validateApplyResumeScheduledScanRequest(req)
	// Add other cases as needed for other request types.
	// ex. we might need be adding suppport for changing schedule for scheduled scans.
	default:
		return errors.Errorf("Cannot validate request of type %T", i)
	}
}

func (m *handlerImpl) processSuspendScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_SuspendScheduledScan) bool {
	config := ScanScheduleConfiguration{
		Suspend:        pointer.Bool(true),
		ScanName:       request.GetScanName(),
		Request:        request,
		ValidationFunc: validateInterface,
	}
	return m.processScanConfigScheduleChangeRequest(requestID, config)
}

func (m *handlerImpl) processResumeScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_ResumeScheduledScan) bool {
	config := ScanScheduleConfiguration{
		Suspend:        pointer.Bool(false),
		ScanName:       request.GetScanName(),
		Request:        request,
		ValidationFunc: validateInterface,
	}
	return m.processScanConfigScheduleChangeRequest(requestID, config)
}
func (m *handlerImpl) processDeleteScanCfgRequest(request *central.DeleteComplianceScanConfigRequest) bool {
	select {
	case <-m.disabled.Done():
		err := errors.Errorf("Compliance is disabled. Cannot process request: %s", protoutils.NewWrapper(request))
		return m.composeAndSendDeleteResponse(request.GetId(), "", err)
	case <-m.stopSignal.Done():
		return true
	default:
		if request.GetName() == defaultScanSettingName {
			return m.composeAndSendDeleteResponse(request.GetId(), "", errors.Errorf("Default compliance scan configuration %q cannot be deleted", defaultScanSettingName))
		}

		// Delete ScanSetting and ScanSettingBinding custom resource. Deleting ScanSettingBindings deletes all owned resources.
		// Each ad-hoc scan creates a unique ScanSettingBinding that reuses default ACS ScanSetting CR.
		// Each scheduled scan configuration has a uniquely identifiable ScanSetting and ScanSettingBinding.
		ns := m.complianceOperatorInfo.GetNamespace()
		if ns == "" {
			return m.composeAndSendDeleteResponse(request.GetId(), "", errors.New("Compliance operator not known"))
		}
		deletePolicy := v1.DeletePropagationForeground
		scanSettingBindingResourceI := m.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(ns)
		err := scanSettingBindingResourceI.Delete(m.ctx(), request.GetName(), v1.DeleteOptions{PropagationPolicy: &deletePolicy})
		if err != nil && !kubeAPIErr.IsNotFound(err) {
			return m.composeAndSendDeleteResponse(request.GetId(), fmt.Sprintf("scansettingbindings/%s", request.GetName()), err)
		}

		scanSettingResourceI := m.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).Namespace(ns)
		err = scanSettingResourceI.Delete(m.ctx(), request.GetName(), v1.DeleteOptions{PropagationPolicy: &deletePolicy})
		if err != nil && !kubeAPIErr.IsNotFound(err) {
			return m.composeAndSendDeleteResponse(request.GetId(), fmt.Sprintf("scansettings/%s", request.GetName()), err)
		}
		return m.composeAndSendDeleteResponse(request.GetId(), "", nil)
	}
}

func (m *handlerImpl) composeAndSendEnableComplianceResponse(requestID string, err error) bool {
	msg := &central.ComplianceResponse{
		Response: &central.ComplianceResponse_EnableComplianceResponse_{
			EnableComplianceResponse: &central.ComplianceResponse_EnableComplianceResponse{
				Id: requestID,
			},
		},
	}
	if err != nil {
		err = errors.Wrap(err, "Could not enable compliance")
		log.Error(err)

		msg.GetEnableComplianceResponse().Payload = &central.ComplianceResponse_EnableComplianceResponse_Error{
			Error: err.Error(),
		}
	}
	return m.sendResponse(msg)
}

func (m *handlerImpl) composeAndSendDisableComplianceResponse(requestID string, err error) bool {
	msg := &central.ComplianceResponse{
		Response: &central.ComplianceResponse_DisableComplianceResponse_{
			DisableComplianceResponse: &central.ComplianceResponse_DisableComplianceResponse{
				Id: requestID,
			},
		},
	}
	if err != nil {
		err = errors.Wrap(err, "Could not enable compliance")
		log.Error(err)

		msg.GetDisableComplianceResponse().Payload = &central.ComplianceResponse_DisableComplianceResponse_Error{
			Error: err.Error(),
		}
	}
	return m.sendResponse(msg)
}

func (m *handlerImpl) composeAndSendApplyScanConfigResponse(requestID string, err error) bool {
	msg := &central.ComplianceResponse{
		Response: &central.ComplianceResponse_ApplyComplianceScanConfigResponse_{
			ApplyComplianceScanConfigResponse: &central.ComplianceResponse_ApplyComplianceScanConfigResponse{
				Id: requestID,
			},
		},
	}
	if err != nil {
		err = errors.Wrap(err, "Could not apply compliance scan configuration")
		log.Error(err)

		msg.GetApplyComplianceScanConfigResponse().Payload = &central.ComplianceResponse_ApplyComplianceScanConfigResponse_Error{
			Error: err.Error(),
		}
	}
	return m.sendResponse(msg)
}

func (m *handlerImpl) composeAndSendDeleteResponse(requestID string, resource string, err error) bool {
	msg := &central.ComplianceResponse{
		Response: &central.ComplianceResponse_DeleteComplianceScanConfigResponse_{
			DeleteComplianceScanConfigResponse: &central.ComplianceResponse_DeleteComplianceScanConfigResponse{
				Id: requestID,
			},
		},
	}
	if err != nil {
		if resource != "" {
			err = errors.Wrapf(err, "Could not delete namespaces/%s/%s", m.complianceOperatorInfo.GetNamespace(), resource)
		}
		log.Error(err)
		msg.GetDeleteComplianceScanConfigResponse().Payload = &central.ComplianceResponse_DeleteComplianceScanConfigResponse_Error{
			Error: err.Error(),
		}
	}
	return m.sendResponse(msg)
}

func (m *handlerImpl) sendResponse(response *central.ComplianceResponse) bool {
	select {
	case m.response <- message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ComplianceResponse{
			ComplianceResponse: response,
		},
	}):
		return true
	case <-m.stopSignal.Done():
		return false
	}
}

func (m *handlerImpl) ctx() context.Context {
	return concurrency.AsContext(&m.stopSignal)
}
