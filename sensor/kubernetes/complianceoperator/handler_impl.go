package complianceoperator

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

type handlerImpl struct {
	ctxLock    sync.Mutex
	streamCtx  context.Context
	response   chan *message.ExpiringMessage
	disabled   concurrency.Signal
	stopSignal concurrency.Signal
	dispatcher common.PubSubDispatcher
	manager    *Manager
}

func (m *handlerImpl) Name() string {
	return "complianceoperator.handlerImpl"
}

// NewRequestHandler returns instance of common.SensorComponent interface which can handle compliance requests from Central.
func NewRequestHandler(manager *Manager, dispatcher common.PubSubDispatcher) common.SensorComponent {
	return &handlerImpl{
		streamCtx:  context.Background(),
		response:   make(chan *message.ExpiringMessage),
		disabled:   concurrency.NewSignal(),
		stopSignal: concurrency.NewSignal(),
		dispatcher: dispatcher,
		manager:    manager,
	}
}

func (m *handlerImpl) Start() error {
	if err := m.dispatcher.RegisterConsumerToLane(
		pubsub.ComplianceHandlerConsumer,
		pubsub.ComplianceCentralResultTopic,
		pubsub.ComplianceCentralResultLane,
		m.onResult,
	); err != nil {
		return err
	}
	return m.manager.Start()
}

func (m *handlerImpl) Stop() {
	// stopSignal is called first to make sure sendResponse is unlocked
	m.stopSignal.Signal()
	m.manager.Stop()
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

func (m *handlerImpl) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	req := msg.GetComplianceRequest()
	if req == nil {
		return nil
	}

	operationName := fmt.Sprintf("%T", req.GetRequest())
	var err error
	ctx := m.getCurrentContext()

	switch r := req.GetRequest().(type) {
	case *central.ComplianceRequest_EnableCompliance:
		m.disabled.Reset()
		m.composeAndSendEnableComplianceResponse(ctx, r.EnableCompliance.GetId(), nil)
	case *central.ComplianceRequest_DisableCompliance:
		m.disabled.Signal()
		m.composeAndSendDisableComplianceResponse(ctx, r.DisableCompliance.GetId(), nil)
	case *central.ComplianceRequest_SyncScanConfigs:
		log.Info("received scan config sync from central")
		err = m.publishRequest(ctx, req)
	case *central.ComplianceRequest_ApplyScanConfig:
		err = m.processApplyScanCfgRequest(ctx, r.ApplyScanConfig, req)
	case *central.ComplianceRequest_DeleteScanConfig:
		err = m.processDeleteScanCfgRequest(ctx, r.DeleteScanConfig, req)
	}

	commandsFromCentral.With(prometheus.Labels{
		"operation": operationName,
		"processed": strconv.FormatBool(err == nil),
	}).Inc()

	return err
}

func (m *handlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return m.response
}

func (m *handlerImpl) SetContext(ctx context.Context) {
	m.ctxLock.Lock()
	defer m.ctxLock.Unlock()
	m.streamCtx = ctx
}

func (m *handlerImpl) getCurrentContext() context.Context {
	m.ctxLock.Lock()
	defer m.ctxLock.Unlock()
	if m.streamCtx == nil {
		return context.Background()
	}
	return m.streamCtx
}

func (m *handlerImpl) onResult(ev pubsub.Event) error {
	result, ok := ev.(*events.ComplianceCentralResultEvent)
	if !ok {
		return errors.Errorf("unexpected event type: %T", ev)
	}
	switch result.Type {
	case events.ApplyScanConfigResponse:
		m.composeAndSendApplyScanConfigResponse(result.Context, result.RequestID, result.Error)
	case events.DeleteScanConfigResponse:
		m.composeAndSendDeleteResponse(result.Context, result.RequestID, result.Error)
	}
	return nil
}

func (m *handlerImpl) processApplyScanCfgRequest(ctx context.Context, request *central.ApplyComplianceScanConfigRequest, fullReq *central.ComplianceRequest) error {
	select {
	case <-m.disabled.Done():
		m.composeAndSendApplyScanConfigResponse(ctx, request.GetId(), errors.Errorf("Compliance is disabled. Cannot process request: %s", protoutils.NewWrapper(request)))
		return nil
	default:
	}

	if request.GetScanRequest() == nil {
		applyScanConfigCommands.WithLabelValues("nil").Inc()
		m.composeAndSendApplyScanConfigResponse(ctx, request.GetId(), errors.New("Compliance scan request is empty"))
		return nil
	}

	applyScanConfigCommands.WithLabelValues(fmt.Sprintf("%T", request.GetScanRequest())).Inc()

	var err error
	switch r := request.GetScanRequest().(type) {
	case *central.ApplyComplianceScanConfigRequest_ScheduledScan_:
		err = validateApplyScheduledScanConfigRequest(r.ScheduledScan)
	case *central.ApplyComplianceScanConfigRequest_UpdateScan:
		err = validateUpdateScheduledScanConfigRequest(r.UpdateScan)
	case *central.ApplyComplianceScanConfigRequest_RerunScan:
		err = validateApplyRerunScheduledScanRequest(r.RerunScan)
	case *central.ApplyComplianceScanConfigRequest_SuspendScan:
		err = validateApplySuspendScheduledScanRequest(r.SuspendScan)
	case *central.ApplyComplianceScanConfigRequest_ResumeScan:
		err = validateApplyResumeScheduledScanRequest(r.ResumeScan)
	default:
		m.composeAndSendApplyScanConfigResponse(ctx, request.GetId(), errors.New("Cannot handle compliance scan request"))
		return nil
	}

	if err != nil {
		m.composeAndSendApplyScanConfigResponse(ctx, request.GetId(), errors.Wrap(err, "validating compliance scan request"))
		return nil
	}

	return m.publishRequest(ctx, fullReq)
}

func (m *handlerImpl) processDeleteScanCfgRequest(ctx context.Context, request *central.DeleteComplianceScanConfigRequest, fullReq *central.ComplianceRequest) error {
	select {
	case <-m.disabled.Done():
		m.composeAndSendDeleteResponse(ctx, request.GetId(), errors.Errorf("Compliance is disabled. Cannot process request: %s", protoutils.NewWrapper(request)))
		return nil
	default:
	}

	if request.GetName() == defaultScanSettingName {
		m.composeAndSendDeleteResponse(ctx, request.GetId(), errors.Errorf("Default compliance scan configuration %q cannot be deleted", defaultScanSettingName))
		return nil
	}

	return m.publishRequest(ctx, fullReq)
}

func (m *handlerImpl) publishRequest(ctx context.Context, req *central.ComplianceRequest) error {
	return m.dispatcher.Publish(&events.ComplianceCentralRequestEvent{Context: ctx, Request: req})
}

func (m *handlerImpl) composeAndSendEnableComplianceResponse(ctx context.Context, requestID string, err error) {
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
	m.sendResponse(ctx, msg)
}

func (m *handlerImpl) composeAndSendDisableComplianceResponse(ctx context.Context, requestID string, err error) {
	msg := &central.ComplianceResponse{
		Response: &central.ComplianceResponse_DisableComplianceResponse_{
			DisableComplianceResponse: &central.ComplianceResponse_DisableComplianceResponse{
				Id: requestID,
			},
		},
	}
	if err != nil {
		err = errors.Wrap(err, "Could not disable compliance")
		log.Error(err)

		msg.GetDisableComplianceResponse().Payload = &central.ComplianceResponse_DisableComplianceResponse_Error{
			Error: err.Error(),
		}
	}
	m.sendResponse(ctx, msg)
}

func (m *handlerImpl) composeAndSendApplyScanConfigResponse(ctx context.Context, requestID string, err error) {
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
	m.sendResponse(ctx, msg)
}

func (m *handlerImpl) composeAndSendDeleteResponse(ctx context.Context, requestID string, err error) {
	msg := &central.ComplianceResponse{
		Response: &central.ComplianceResponse_DeleteComplianceScanConfigResponse_{
			DeleteComplianceScanConfigResponse: &central.ComplianceResponse_DeleteComplianceScanConfigResponse{
				Id: requestID,
			},
		},
	}
	if err != nil {
		log.Error(err)
		msg.GetDeleteComplianceScanConfigResponse().Payload = &central.ComplianceResponse_DeleteComplianceScanConfigResponse_Error{
			Error: err.Error(),
		}
	}
	m.sendResponse(ctx, msg)
}

func (m *handlerImpl) sendResponse(ctx context.Context, response *central.ComplianceResponse) {
	select {
	case m.response <- message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ComplianceResponse{
			ComplianceResponse: response,
		},
	}):
	case <-m.stopSignal.Done():
	case <-ctx.Done():
	}
}
