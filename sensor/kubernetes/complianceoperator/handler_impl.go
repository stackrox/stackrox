package complianceoperator

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/pointer"
)

const (
	defaultMaxRetries     = 5
	defaultAPICallTimeout = 5 * time.Second
	defaultRetryTimeout   = 30 * time.Second
)

type handlerImpl struct {
	complianceOperatorInfo StatusInfo

	response chan *message.ExpiringMessage
	request  chan *central.ComplianceRequest

	disabled          concurrency.Signal
	stopSignal        concurrency.Signal
	started           *atomic.Bool
	complianceIsReady *concurrency.Signal
	retrier           *retrier
	writer            *scanSettingWriter
}

func (m *handlerImpl) Name() string {
	return "complianceoperator.handlerImpl"
}

type scanScheduleConfiguration struct {
	Suspend  *bool
	Schedule *string
	ScanName string
}

// NewRequestHandler returns instance of common.SensorComponent interface which can handle compliance requests from Central.
func NewRequestHandler(client dynamic.Interface, complianceOperatorInfo StatusInfo, coIsReady *concurrency.Signal) common.SensorComponent {
	r := &retrier{
		maxRetries:     defaultMaxRetries,
		apiCallTimeout: defaultAPICallTimeout,
		retryTimeout:   defaultRetryTimeout,
	}
	w := &scanSettingWriter{
		client:  client,
		retrier: r,
	}
	return &handlerImpl{
		complianceOperatorInfo: complianceOperatorInfo,

		request:  make(chan *central.ComplianceRequest),
		response: make(chan *message.ExpiringMessage),

		started:           &atomic.Bool{},
		disabled:          concurrency.NewSignal(),
		stopSignal:        concurrency.NewSignal(),
		complianceIsReady: coIsReady,
		retrier:           r,
		writer:            w,
	}
}

func (m *handlerImpl) Start() error {
	defer m.started.Store(true)
	// TODO: create default scan setting for ad-hoc scan
	go m.run()
	return nil
}

func (m *handlerImpl) Stop() {
	m.stopSignal.Signal()
}

func (m *handlerImpl) Notify(_ common.SensorComponentEvent) {}

func (m *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	if syncScanConfigsOnStartup.BooleanSetting() {
		return []centralsensor.SensorCapability{centralsensor.ComplianceV2ScanConfigSync}
	}
	return nil
}

func (m *handlerImpl) Accepts(msg *central.MsgToSensor) bool {
	return msg.GetComplianceRequest() != nil
}

func (m *handlerImpl) ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error {
	req := msg.GetComplianceRequest()
	if req == nil {
		return nil
	}
	// Sync Scan Configs is done during the syncing process between Sensor and Central
	if _, ok := req.GetRequest().(*central.ComplianceRequest_SyncScanConfigs); ok {
		log.Info("received scan config sync from central")
		return m.handleSyncScanCfgRequest(req.GetSyncScanConfigs())
	}

	if !m.started.Load() {
		return errors.Errorf("the compliance operator handler was not started ignoring message %v to avoid blocking", msg.GetComplianceRequest())
	}

	select {
	case <-ctx.Done():
		// TODO(ROX-30333): Pass this context together with `req` to `m.request`
		return errors.Wrapf(ctx.Err(), "message processing in component %s", m.Name())
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
			operationName := fmt.Sprintf("%T", req.GetRequest())
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
			commandsFromCentral.With(prometheus.Labels{
				"operation": operationName,
				"processed": strconv.FormatBool(requestProcessed)},
			).Inc()

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
			applyScanConfigCommands.WithLabelValues("nil").Inc()
			return m.composeAndSendApplyScanConfigResponse(request.GetId(), errors.New("Compliance scan request is empty"))
		}

		applyScanConfigCommands.WithLabelValues(fmt.Sprintf("%T", request.GetScanRequest())).Inc()
		switch r := request.GetScanRequest().(type) {
		case *central.ApplyComplianceScanConfigRequest_ScheduledScan_:
			return m.processScheduledScanRequest(request.GetId(), r.ScheduledScan)
		case *central.ApplyComplianceScanConfigRequest_SuspendScan:
			return m.processSuspendScheduledScanRequest(request.GetId(), r.SuspendScan)
		case *central.ApplyComplianceScanConfigRequest_ResumeScan:
			return m.processResumeScheduledScanRequest(request.GetId(), r.ResumeScan)
		case *central.ApplyComplianceScanConfigRequest_RerunScan:
			return m.processRerunScheduledScanRequest(request.GetId(), r.RerunScan)
		case *central.ApplyComplianceScanConfigRequest_UpdateScan:
			return m.processUpdateScanRequest(request.GetId(), r.UpdateScan)
		default:
			return m.composeAndSendApplyScanConfigResponse(request.GetId(), errors.New("Cannot handle compliance scan request"))
		}
	}
}

func (m *handlerImpl) processScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_ScheduledScan) bool {
	if err := validateApplyScheduledScanConfigRequest(request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}
	err := m.writer.createScanResources(m.ctx(), ns, request.GetScanSettings(), request.GetCron())
	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}

func (m *handlerImpl) processUpdateScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_UpdateScheduledScan) bool {
	if err := validateUpdateScheduledScanConfigRequest(request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}
	err := m.writer.updateScanConfig(m.ctx(), ns, request)
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
	err := m.writer.rerunScan(m.ctx(), ns, request.GetScanName())
	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}

func (m *handlerImpl) processSuspendScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_SuspendScheduledScan) bool {
	if err := validateApplySuspendScheduledScanRequest(request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}
	config := scanScheduleConfiguration{
		Suspend:  pointer.Bool(true),
		ScanName: request.GetScanName(),
	}
	err := m.writer.changeScanSchedule(m.ctx(), ns, config)
	return m.composeAndSendApplyScanConfigResponse(requestID, err)
}

func (m *handlerImpl) processResumeScheduledScanRequest(requestID string, request *central.ApplyComplianceScanConfigRequest_ResumeScheduledScan) bool {
	if err := validateApplyResumeScheduledScanRequest(request); err != nil {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.Wrap(err, "validating compliance scan request"))
	}
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return m.composeAndSendApplyScanConfigResponse(requestID, errors.New("Compliance operator namespace not known"))
	}
	config := scanScheduleConfiguration{
		Suspend:  pointer.Bool(false),
		ScanName: request.GetScanName(),
	}
	err := m.writer.changeScanSchedule(m.ctx(), ns, config)
	return m.composeAndSendApplyScanConfigResponse(requestID, err)
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
		ns := m.complianceOperatorInfo.GetNamespace()
		if ns == "" {
			return m.composeAndSendDeleteResponse(request.GetId(), "", errors.New("Compliance operator not known"))
		}
		err := m.writer.deleteScanConfig(m.ctx(), ns, request.GetName())
		return m.composeAndSendDeleteResponse(request.GetId(), "", err)
	}
}

func (m *handlerImpl) handleSyncScanCfgRequest(request *central.SyncComplianceScanConfigRequest) error {
	go func() {
		select {
		case <-m.disabled.Done():
			log.Errorf("Compliance is disabled. Cannot process request: %s", protoutils.NewWrapper(request))
			return
		case <-m.stopSignal.Done():
			return
		case <-m.complianceIsReady.Done():
			ns := m.complianceOperatorInfo.GetNamespace()
			if ns == "" {
				log.Error("Compliance operator namespace not known")
				return
			}
			log.Debugf("compliance is ready. Starting the reconciliation of %d scan configs", len(request.GetScanConfigs()))
			if err := m.writer.syncScanConfigs(m.ctx(), ns, request); err != nil {
				log.Error(err)
			}
		}
	}()
	return nil
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
