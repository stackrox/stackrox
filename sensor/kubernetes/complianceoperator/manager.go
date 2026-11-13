package complianceoperator

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/noderole/store"
	"k8s.io/client-go/dynamic"
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
		return errors.Wrap(err, "registering central request consumer")
	}
	if err := m.dispatcher.RegisterConsumerToLane(
		pubsub.ComplianceManagerConsumer,
		pubsub.NodeRoleChangeTopic,
		pubsub.NodeRoleChangeLane,
		m.onMCPEvent,
	); err != nil {
		return errors.Wrap(err, "registering node role change consumer")
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

func (m *Manager) processSync(ctx context.Context, request *central.SyncComplianceScanConfigRequest) error {
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		return errors.New("compliance operator namespace not known")
	}
	mergedCtx, stopAfterFn := m.mergeContextWithStopper(ctx)
	defer func() {
		_ = stopAfterFn()
	}()
	return m.writer.syncScanConfigs(mergedCtx, ns, request)
}

func (m *Manager) process(ev pubsub.Event) {
	e, ok := ev.(*events.ComplianceCentralRequestEvent)
	if !ok {
		log.Errorf("unexpected event type: %T", ev)
		return
	}
	select {
	case <-e.Context.Done():
		return
	default:
	}
	req := e.Request
	switch r := req.GetRequest().(type) {
	case *central.ComplianceRequest_ApplyScanConfig:
		m.processApplyScanConfig(e.Context, r.ApplyScanConfig)
	case *central.ComplianceRequest_DeleteScanConfig:
		m.processDeleteScanConfig(e.Context, r.DeleteScanConfig)
	case *central.ComplianceRequest_SyncScanConfigs:
		if err := m.processSync(e.Context, r.SyncScanConfigs); err != nil {
			log.Errorf("Failed to process scan config sync: %v", err)
		}
	default:
		log.Errorf("unexpected request type in manager: %T", req.GetRequest())
	}
}

func (m *Manager) processApplyScanConfig(ctx context.Context, request *central.ApplyComplianceScanConfigRequest) {
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		m.publishResult(ctx, request.GetId(), events.ApplyScanConfigResponse, errors.New("compliance operator namespace not known"))
		return
	}

	mergedCtx, stopAfterFn := m.mergeContextWithStopper(ctx)
	defer func() {
		_ = stopAfterFn()
	}()

	var err error
	switch r := request.GetScanRequest().(type) {
	case *central.ApplyComplianceScanConfigRequest_ScheduledScan_:
		err = m.writer.createScanResources(mergedCtx, ns, r.ScheduledScan.GetScanSettings(), r.ScheduledScan.GetCron())
	case *central.ApplyComplianceScanConfigRequest_UpdateScan:
		err = m.writer.updateScanConfig(mergedCtx, ns, r.UpdateScan)
	case *central.ApplyComplianceScanConfigRequest_RerunScan:
		err = m.writer.rerunScan(mergedCtx, ns, r.RerunScan.GetScanName())
	case *central.ApplyComplianceScanConfigRequest_SuspendScan:
		config := scanScheduleConfiguration{
			Suspend:  new(true),
			ScanName: r.SuspendScan.GetScanName(),
		}
		err = m.writer.changeScanSchedule(mergedCtx, ns, config)
	case *central.ApplyComplianceScanConfigRequest_ResumeScan:
		config := scanScheduleConfiguration{
			Suspend:  new(false),
			ScanName: r.ResumeScan.GetScanName(),
		}
		err = m.writer.changeScanSchedule(mergedCtx, ns, config)
	default:
		err = errors.Errorf("unknown scan request type: %T", request.GetScanRequest())
	}

	m.publishResult(ctx, request.GetId(), events.ApplyScanConfigResponse, err)
}

func (m *Manager) processDeleteScanConfig(ctx context.Context, request *central.DeleteComplianceScanConfigRequest) {
	// Delete ScanSetting and ScanSettingBinding custom resource. Deleting ScanSettingBindings deletes all owned resources.
	// Each ad-hoc scan creates a unique ScanSettingBinding that reuses default ACS ScanSetting CR.
	// Each scheduled scan configuration has a uniquely identifiable ScanSetting and ScanSettingBinding.
	ns := m.complianceOperatorInfo.GetNamespace()
	if ns == "" {
		m.publishResult(ctx, request.GetId(), events.DeleteScanConfigResponse, errors.New("compliance operator namespace not known"))
		return
	}
	mergedCtx, stopAfterFn := m.mergeContextWithStopper(ctx)
	defer func() {
		_ = stopAfterFn()
	}()
	err := m.writer.deleteScanConfig(mergedCtx, ns, request.GetName())
	m.publishResult(ctx, request.GetId(), events.DeleteScanConfigResponse, err)
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

	if err := m.writer.updateAllScanSettingRoles(m.stopperContext(), ns, m.getRoles()); err != nil {
		log.Errorf("failed to reconcile MCP roles: %v", err)
	}
}

func (m *Manager) publishResult(ctx context.Context, requestID string, responseType events.ResponseType, err error) {
	if pubErr := m.dispatcher.Publish(&events.ComplianceCentralResultEvent{
		Context:   ctx,
		RequestID: requestID,
		Error:     err,
		Type:      responseType,
	}); pubErr != nil {
		log.Errorf("failed to publish compliance result for request %s: %v", requestID, pubErr)
	}
}

func (m *Manager) stopperContext() context.Context {
	return concurrency.AsContext(concurrency.WaitableChan(m.stopper.Flow().StopRequested()))
}

func (m *Manager) mergeContextWithStopper(ctx context.Context) (context.Context, func() bool) {
	return contextutil.MergeContext(ctx, m.stopperContext())
}
