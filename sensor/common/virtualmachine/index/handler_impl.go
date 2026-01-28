package index

import (
	"context"
	"maps"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
)

var (
	errCapabilityNotSupported = errors.New("Central does not have virtual machine capability")
	errCentralNotReachable    = errors.New("Central is not reachable")
	errInputChanClosed        = errors.New("channel receiving virtual machines is closed")
	errStartMoreThanOnce      = errors.New("unable to start the handler more than once")
	errVirtualMachineNotFound = errors.New("virtual machine not found")
)

type handlerImpl struct {
	centralReady concurrency.Signal
	// lock prevents the race condition between Start() [writer] and ResponsesC(), Send() [reader].
	lock         *sync.RWMutex
	stopper      concurrency.Stopper
	toCentral    <-chan *message.ExpiringMessage
	indexReports chan *indexReportWithData
	clusterID    clusterIDGetter
	store        VirtualMachineStore
}

type indexReportWithData struct {
	report         *v1.IndexReport
	discoveredData *v1.DiscoveredData
}

func (h *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *handlerImpl) Send(ctx context.Context, vm *v1.IndexReport, discoveredData *v1.DiscoveredData) error {
	if h.stopper.Client().Stopped().IsDone() {
		return errox.InvariantViolation.CausedBy(errInputChanClosed)
	}
	if !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
		return errox.NotImplemented.CausedBy(errCapabilityNotSupported)
	}
	h.lock.RLock()
	defer h.lock.RUnlock()
	if h.indexReports == nil {
		return errox.InvariantViolation.CausedBy(errInputChanClosed)
	}
	if !h.centralReady.IsDone() {
		log.Warnf("Cannot send index report for virtual machine with vsock_cid=%q to Central because Central is not reachable", vm.GetVsockCid())
		metrics.IndexReportsSent.With(metrics.StatusCentralNotReadyLabels).Inc()
		return errox.ResourceExhausted.CausedBy(errCentralNotReachable)
	}

	blockingStart := time.Now()
	blocked := false
	outcome := metrics.IndexReportEnqueueOutcomeSuccess
	defer func() {
		if blocked {
			metrics.IndexReportBlockingEnqueueDurationMilliseconds.
				WithLabelValues(outcome).
				Observe(metrics.StartTimeToMS(blockingStart))
		}
	}()

	item := &indexReportWithData{
		report:         vm,
		discoveredData: discoveredData,
	}

	// Fast-path select to detect blocking on the channel for metrics
	// Check context first to prioritize cancellation
	select {
	case <-ctx.Done():
		// Context canceled, handle in next select statement
	case h.indexReports <- item:
		// Double-check context wasn't canceled during send
		select {
		case <-ctx.Done():
			// Context was canceled, return error immediately
			if err := ctx.Err(); errors.Is(err, context.DeadlineExceeded) {
				outcome = metrics.IndexReportEnqueueOutcomeTimeout
				return err //nolint:wrapcheck
			}
			outcome = metrics.IndexReportEnqueueOutcomeCanceled
			return ctx.Err() //nolint:wrapcheck
		default:
			return nil
		}
	default:
		blocked = true
		blockingStart = time.Now()
		metrics.IndexReportEnqueueBlockedTotal.Inc()
	}

	select {
	case <-ctx.Done():
		if err := ctx.Err(); errors.Is(err, context.DeadlineExceeded) {
			outcome = metrics.IndexReportEnqueueOutcomeTimeout
			return err //nolint:wrapcheck
		}
		outcome = metrics.IndexReportEnqueueOutcomeCanceled
		return ctx.Err() //nolint:wrapcheck
	case h.indexReports <- item:
		return nil
	}
}

func (h *handlerImpl) Name() string {
	return "virtualmachine.index.handlerImpl"
}

func (h *handlerImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		h.centralReady.Signal()
	case common.SensorComponentEventOfflineMode:
		// As clients are expected to retry virtual machine upserts when Sensor is in
		// offline mode, there is no need to do anything here other than reset the signal.
		h.centralReady.Reset()
	}
}

func (h *handlerImpl) Accepts(msg *central.MsgToSensor) bool {
	if sensorAck := msg.GetSensorAck(); sensorAck != nil {
		return sensorAck.GetMessageType() == central.SensorACK_VM_INDEX_REPORT
	}
	return false
}

// ProcessMessage handles SensorACK messages for VM index reports.
func (h *handlerImpl) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	sensorAck := msg.GetSensorAck()
	if sensorAck == nil || sensorAck.GetMessageType() != central.SensorACK_VM_INDEX_REPORT {
		return nil
	}

	vmID := sensorAck.GetResourceId()
	action := sensorAck.GetAction()
	reason := sensorAck.GetReason()

	switch action {
	case central.SensorACK_ACK:
		log.Debugf("Received ACK from Central for VM index report: vm_id=%s", vmID)
		metrics.IndexReportAcksReceived.WithLabelValues(action.String()).Inc()
	case central.SensorACK_NACK:
		log.Warnf("Received NACK from Central for VM index report: vm_id=%s, reason=%s", vmID, reason)
		metrics.IndexReportAcksReceived.WithLabelValues(action.String()).Inc()
		// TODO(ROX-xxxxx): Implement retry logic or notifying VM relay.
		// Currently, the VM relay has its own retry mechanism, but it's not aware of Central's rate limiting.
	}

	return nil
}

// ResponsesC returns a channel with messages to Central. It must be called
// after Start() for the channel to be not nil.
func (h *handlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	h.lock.RLock()
	defer h.lock.RUnlock()
	if h.toCentral == nil {
		log.Panic("Start must be called before ResponsesC")
	}
	return h.toCentral
}

func (h *handlerImpl) Start() error {
	log.Debug("Starting virtual machine handler")
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.toCentral != nil || h.indexReports != nil {
		return errStartMoreThanOnce
	}
	h.indexReports = make(chan *indexReportWithData, env.VirtualMachinesIndexReportsBufferSize.IntegerSetting())
	h.toCentral = h.run(h.indexReports)
	return nil
}

func (h *handlerImpl) Stop() {
	// Stop the stopper FIRST so Send() will see it as stopped and return early
	// before we close the channel. This prevents panics from sending on closed channel.
	// Matters mainly for local-sensor, as we care that local-sensor stops cleanly before saving the data to a file.
	client := h.stopper.Client()
	if !client.Stopped().IsDone() {
		defer utils.IgnoreError(client.Stopped().Wait)
		client.Stop()
	}
	// Acquire write lock to prevent concurrent Send() calls from racing with channel close
	h.lock.Lock()
	defer h.lock.Unlock()
	// Now close the channel - this will cause the run() goroutine to exit.
	// Guard against closing an already-closed channel to make Stop() idempotent
	if h.indexReports != nil {
		close(h.indexReports)
		h.indexReports = nil
	}
}

// run handles the virtual machine data and forwards it to Central.
// This is the only goroutine that writes into the toCentral channel, thus it is
// responsible for creating and closing that chan.
func (h *handlerImpl) run(
	indexReports <-chan *indexReportWithData,
) (toCentral <-chan *message.ExpiringMessage) {
	ch2Central := make(chan *message.ExpiringMessage)
	go func() {
		defer func() {
			h.stopper.Flow().ReportStopped()
			close(ch2Central)
		}()
		log.Debugf("virtual machine index report handler is running")
		for {
			select {
			case <-h.stopper.Flow().StopRequested():
				return
			case item, ok := <-indexReports:
				if !ok {
					h.stopper.Flow().StopWithError(errInputChanClosed)
					return
				}
				h.handleIndexReport(ch2Central, item)
			}
		}
	}()
	return ch2Central
}

func (h *handlerImpl) handleIndexReport(
	toCentral chan *message.ExpiringMessage,
	item *indexReportWithData,
) {
	startTime := time.Now()
	outcome := metrics.IndexReportHandlingMessageToCentralSuccess
	defer func() {
		metrics.IndexReportProcessingDurationMilliseconds.
			WithLabelValues(outcome).
			Observe(metrics.StartTimeToMS(startTime))
	}()

	if item == nil || item.report == nil {
		outcome = metrics.IndexReportHandlingMessageToCentralNilReport
		log.Warn("Received nil virtual machine index report: not sending to Central")
		return
	}
	indexReport := item.report
	log.Debugf("Handling virtual machine index report with vsock_cid=%q...", indexReport.GetVsockCid())

	// Handle discovered facts if feature is enabled and we have discovered data
	if features.VirtualMachines.Enabled() && item.discoveredData != nil {
		h.handleDiscoveredFacts(toCentral, indexReport, item.discoveredData)
	}

	msg, outcome, err := h.newMessageToCentral(indexReport)
	if err != nil {
		// TODO: send a message the sensor relay to retry later if the VM was not found
		log.Warnf("unable to send index report message for the virtual machine with vsock cid %q to central: %v", indexReport.GetVsockCid(), err)
		return
	}
	h.sendToCentral(toCentral, msg)
	metrics.IndexReportsSent.With(metrics.StatusSuccessLabels).Inc()
}

func (h *handlerImpl) handleDiscoveredFacts(
	toCentral chan *message.ExpiringMessage,
	indexReport *v1.IndexReport,
	data *v1.DiscoveredData,
) {
	if data == nil {
		return
	}

	cid, err := strconv.ParseUint(indexReport.GetVsockCid(), 10, 32)
	if err != nil {
		log.Warnf("Invalid vsock CID %q in index report: %v", indexReport.GetVsockCid(), err)
		return
	}

	vmInfo := h.store.GetFromCID(uint32(cid))
	if vmInfo == nil {
		log.Debugf("VM with vsock_cid=%q not found, skipping discovered facts storage", indexReport.GetVsockCid())
		metrics.IndexReportsForUnknownVMCID.Inc()
		return
	}

	facts := factsFromDiscoveredData(data)
	if len(facts) == 0 {
		return
	}

	previousFacts := h.store.GetDiscoveredFacts(vmInfo.ID)
	h.store.UpsertDiscoveredFacts(vmInfo.ID, facts)

	if !maps.Equal(previousFacts, facts) {
		// Emit VM update message when facts change
		msg := h.newVirtualMachineUpdateMessage(vmInfo)
		h.sendToCentral(toCentral, msg)
	}
}

func factsFromDiscoveredData(data *v1.DiscoveredData) map[string]string {
	facts := make(map[string]string)
	if data.GetDetectedOs() != v1.DetectedOS_UNKNOWN {
		facts[virtualmachine.FactsDetectedOSKey] = data.GetDetectedOs().String()
	}
	if data.GetOsVersion() != "" {
		facts[virtualmachine.FactsOSVersionKey] = data.GetOsVersion()
	}
	if data.GetActivationStatus() != v1.ActivationStatus_ACTIVATION_UNSPECIFIED {
		facts[virtualmachine.FactsActivationStatusKey] = data.GetActivationStatus().String()
	}
	if data.GetDnfMetadataStatus() != v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED {
		facts[virtualmachine.FactsDNFMetadataStatusKey] = data.GetDnfMetadataStatus().String()
	}
	return facts
}

func (h *handlerImpl) newMessageToCentral(indexReport *v1.IndexReport) (*message.ExpiringMessage, string, error) {
	cid, err := strconv.ParseUint(indexReport.GetVsockCid(), 10, 32)
	if err != nil {
		return nil, metrics.IndexReportHandlingMessageToCentralInvalidCID, errors.Wrapf(err, "Received an invalid Vsock CID: %q", indexReport.GetVsockCid())
	}

	vmInfo := h.store.GetFromCID(uint32(cid))
	if vmInfo == nil {
		// Return retryable error if the virtual machine is not yet known to Sensor.
		return nil, metrics.IndexReportHandlingMessageToCentralVMUnknown, errors.Wrapf(errVirtualMachineNotFound, "VirtualMachine with Vsock CID %q not found", indexReport.GetVsockCid())
	}

	return message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     string(vmInfo.ID),
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachineIndexReport{
					VirtualMachineIndexReport: &v1.IndexReportEvent{
						Id:    string(vmInfo.ID),
						Index: indexReport,
					},
				},
			},
		},
	}), metrics.IndexReportHandlingMessageToCentralSuccess, nil
}

func (h *handlerImpl) newVirtualMachineUpdateMessage(vmInfo *virtualmachine.Info) *message.ExpiringMessage {
	clusterID := ""
	if h.clusterID != nil {
		clusterID = h.clusterID.Get()
	}
	vSockCID, vSockCIDSet := virtualmachine.VSockCIDFromInfo(vmInfo)
	return message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     string(vmInfo.ID),
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &v1.VirtualMachine{
						Id:          string(vmInfo.ID),
						Namespace:   vmInfo.Namespace,
						Name:        vmInfo.Name,
						ClusterId:   clusterID,
						VsockCid:    vSockCID,
						VsockCidSet: vSockCIDSet,
						State:       virtualmachine.StateFromInfo(vmInfo),
						Facts:       virtualmachine.BuildFacts(vmInfo, h.store.GetDiscoveredFacts(vmInfo.ID)),
					},
				},
			},
		},
	})
}

func (h *handlerImpl) sendToCentral(
	toCentral chan<- *message.ExpiringMessage,
	msg *message.ExpiringMessage,
) {
	// The `toCentral` is closed in the same goroutine, so it will be still open when stop is requested.
	select {
	case <-h.stopper.Flow().StopRequested():
	case toCentral <- msg:
	}
}
