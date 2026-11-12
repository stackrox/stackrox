package complianceoperator

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/noderole/store"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"
)

// Manager is the sole writer of ScanSetting and ScanSettingBinding
// resources to the Kubernetes API. It multiplexes Central requests
// and MCP change events into a single goroutine to avoid races.
type Manager struct {
	complianceIsReady      *concurrency.Signal
	complianceOperatorInfo StatusInfo
	mcpStore               *store.MachineConfigPoolStore
	stopper                concurrency.Stopper
	dispatcher             common.PubSubDispatcher
	inner                  chan pubsub.Event
	mcpNotify              chan struct{}
	mcpDebounceInterval    time.Duration
	writer                 *scanSettingWriter
}

// NewManager creates a new Manager.
func NewManager(client dynamic.Interface, complianceOperatorInfo StatusInfo, mcpStore *store.MachineConfigPoolStore, complianceIsReady *concurrency.Signal, dispatcher common.PubSubDispatcher) *Manager {
	r := &retrier{
		maxRetries:     defaultMaxRetries,
		apiCallTimeout: defaultAPICallTimeout,
		retryTimeout:   defaultRetryTimeout,
	}
	m := &Manager{
		complianceIsReady:      complianceIsReady,
		complianceOperatorInfo: complianceOperatorInfo,
		mcpStore:               mcpStore,
		stopper:                concurrency.NewStopper(),
		dispatcher:             dispatcher,
		inner:                  make(chan pubsub.Event),
		mcpNotify:              make(chan struct{}, 1),
		mcpDebounceInterval:    defaultMCPDebounceInterval,
		writer:                 &scanSettingWriter{client: client, retrier: r},
	}
	m.writer.roleProvider = m.getRoles
	return m
}

func (m *Manager) getRoles() []string {
	if features.ComplianceCustomNodeRoles.Enabled() && m.mcpStore != nil {
		roles := m.mcpStore.GetRoles()
		if len(roles) > 0 {
			return roles
		}
	}
	return []string{masterRole, workerRole}
}

// Start registers the pubsub consumers and launches the run loop.
func (m *Manager) Start() error {
	if err := m.dispatcher.RegisterConsumerToLane(
		pubsub.ComplianceManagerConsumer,
		pubsub.ComplianceCentralRequestTopic,
		pubsub.ComplianceCentralRequestLane,
		m.onCentralRequest,
	); err != nil {
		return err
	}
	if err := m.dispatcher.RegisterConsumerToLane(
		pubsub.ComplianceManagerConsumer,
		pubsub.NodeRoleChangeTopic,
		pubsub.NodeRoleChangeLane,
		m.onMCPEvent,
	); err != nil {
		return err
	}
	go m.run()
	return nil
}

// Stop requests shutdown and waits for the run loop to finish.
func (m *Manager) Stop() {
	if !m.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = m.stopper.Client().Stopped().Wait()
		}()
	}
	m.stopper.Client().Stop()
}

func (m *Manager) onCentralRequest(ev pubsub.Event) error {
	select {
	case m.inner <- ev:
		return nil
	case <-m.stopper.Flow().StopRequested():
		return nil
	}
}

func (m *Manager) onMCPEvent(_ pubsub.Event) error {
	select {
	case m.mcpNotify <- struct{}{}:
	default:
	}
	return nil
}

func (m *Manager) run() {
	defer m.stopper.Flow().ReportStopped()

	select {
	case <-m.complianceIsReady.Done():
	case <-m.stopper.Flow().StopRequested():
		return
	}

	if centralcaps.Has(centralsensor.ComplianceV2ScanConfigSync) {
		m.waitAndProcessSync()
	}

	var mcpTimer *time.Timer
	var mcpTimerC <-chan time.Time

	for {
		select {
		case ev := <-m.inner:
			m.process(ev)
		case <-m.mcpNotify:
			if mcpTimer != nil {
				mcpTimer.Stop()
			}
			mcpTimer = time.NewTimer(m.mcpDebounceInterval)
			mcpTimerC = mcpTimer.C
		case <-mcpTimerC:
			mcpTimerC = nil
			m.reconcileMCPRoles()
		case <-m.stopper.Flow().StopRequested():
			if mcpTimer != nil {
				mcpTimer.Stop()
			}
			return
		}
	}
}

func (m *Manager) waitAndProcessSync() {
	select {
	case ev := <-m.inner:
		if err := m.processSync(ev); err != nil {
			log.Errorf("Failed to process scan config sync: %v", err)
		}
	case <-m.stopper.Flow().StopRequested():
	}
}

func (m *Manager) processSync(ev pubsub.Event) error {
	e, ok := ev.(*events.ComplianceCentralRequestEvent)
	if !ok {
		return errors.Errorf("unexpected event type: %T", ev)
	}
	syncReq := e.Request.GetSyncScanConfigs()
	if syncReq == nil {
		return errors.New("expected SyncScanConfigs request")
	}
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return errors.New("compliance operator namespace not known")
	}
	return m.writer.syncScanConfigs(m.ctx(), ns, syncReq)
}

func (m *Manager) process(ev pubsub.Event) {
	e, ok := ev.(*events.ComplianceCentralRequestEvent)
	if !ok {
		log.Errorf("unexpected event type: %T", ev)
		return
	}
	req := e.Request
	switch r := req.GetRequest().(type) {
	case *central.ComplianceRequest_ApplyScanConfig:
		m.processApplyScanConfig(r.ApplyScanConfig)
	case *central.ComplianceRequest_DeleteScanConfig:
		m.processDeleteScanConfig(r.DeleteScanConfig)
	default:
		log.Errorf("unexpected request type in manager: %T", req.GetRequest())
	}
}

func (m *Manager) processApplyScanConfig(request *central.ApplyComplianceScanConfigRequest) {
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		m.publishResult(request.GetId(), events.ApplyScanConfigResponse, errors.New("compliance operator namespace not known"))
		return
	}

	var err error
	switch r := request.GetScanRequest().(type) {
	case *central.ApplyComplianceScanConfigRequest_ScheduledScan_:
		err = m.writer.createScanResources(m.ctx(), ns, r.ScheduledScan.GetScanSettings(), r.ScheduledScan.GetCron())
	case *central.ApplyComplianceScanConfigRequest_UpdateScan:
		err = m.writer.updateScanConfig(m.ctx(), ns, r.UpdateScan)
	case *central.ApplyComplianceScanConfigRequest_RerunScan:
		err = m.writer.rerunScan(m.ctx(), ns, r.RerunScan.GetScanName())
	case *central.ApplyComplianceScanConfigRequest_SuspendScan:
		config := scanScheduleConfiguration{
			Suspend:  ptr.To(true),
			ScanName: r.SuspendScan.GetScanName(),
		}
		err = m.writer.changeScanSchedule(m.ctx(), ns, config)
	case *central.ApplyComplianceScanConfigRequest_ResumeScan:
		config := scanScheduleConfiguration{
			Suspend:  ptr.To(false),
			ScanName: r.ResumeScan.GetScanName(),
		}
		err = m.writer.changeScanSchedule(m.ctx(), ns, config)
	default:
		err = errors.Errorf("unknown scan request type: %T", request.GetScanRequest())
	}

	m.publishResult(request.GetId(), events.ApplyScanConfigResponse, err)
}

func (m *Manager) processDeleteScanConfig(request *central.DeleteComplianceScanConfigRequest) {
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		m.publishResult(request.GetId(), events.DeleteScanConfigResponse, errors.New("compliance operator namespace not known"))
		return
	}
	err := m.writer.deleteScanConfig(m.ctx(), ns, request.GetName())
	m.publishResult(request.GetId(), events.DeleteScanConfigResponse, err)
}

func (m *Manager) reconcileMCPRoles() {
	if !features.ComplianceCustomNodeRoles.Enabled() {
		return
	}
	if m.mcpStore == nil {
		return
	}
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		log.Error("compliance operator namespace not known, skipping MCP role reconciliation")
		return
	}
	roles := m.mcpStore.GetRoles()
	if len(roles) == 0 {
		log.Debug("no MCP roles found, skipping reconciliation")
		return
	}
	if err := m.writer.updateAllScanSettingRoles(m.ctx(), ns, roles); err != nil {
		log.Errorf("failed to reconcile MCP roles: %v", err)
	}
}

func (m *Manager) publishResult(requestID string, responseType events.ResponseType, err error) {
	if pubErr := m.dispatcher.Publish(&events.ComplianceCentralResultEvent{
		RequestID: requestID,
		Error:     err,
		Type:      responseType,
	}); pubErr != nil {
		log.Errorf("failed to publish compliance result for request %s: %v", requestID, pubErr)
	}
}

func (m *Manager) ctx() context.Context {
	return concurrency.AsContext(concurrency.WaitableChan(m.stopper.Flow().StopRequested()))
}
