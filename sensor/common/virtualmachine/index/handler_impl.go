package index

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
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
	indexReports chan *v1.IndexReport
	store        VirtualMachineStore
}

func (h *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *handlerImpl) Send(ctx context.Context, vm *v1.IndexReport) error {
	if h.stopper.Client().Stopped().IsDone() {
		return errox.InvariantViolation.CausedBy(errInputChanClosed)
	}
	if !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
		return errox.NotImplemented.CausedBy(errCapabilityNotSupported)
	}
	if !h.centralReady.IsDone() {
		log.Warnf("Cannot send index report for virtual machine with vsock_cid=%q to Central because Central is not reachable", vm.GetVsockCid())
		metrics.IndexReportsSent.With(metrics.StatusCentralNotReadyLabels).Inc()
		return errox.ResourceExhausted.CausedBy(errCentralNotReachable)
	}

	h.lock.RLock()
	defer h.lock.RUnlock()

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

	// Fast-path select to detect blocking on the channel for metrics
	select {
	case <-ctx.Done():
		// Handled in the next select statement
	case h.indexReports <- vm:
		return nil
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
	case h.indexReports <- vm:
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
	h.indexReports = make(chan *v1.IndexReport, env.VirtualMachinesIndexReportsBufferSize.IntegerSetting())
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
func (h *handlerImpl) run(indexReports <-chan *v1.IndexReport) (toCentral <-chan *message.ExpiringMessage) {
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
			case indexReport, ok := <-indexReports:
				if !ok {
					h.stopper.Flow().StopWithError(errInputChanClosed)
					return
				}
				h.handleIndexReport(ch2Central, indexReport)
			}
		}
	}()
	return ch2Central
}

func (h *handlerImpl) handleIndexReport(
	toCentral chan *message.ExpiringMessage,
	indexReport *v1.IndexReport,
) {
	startTime := time.Now()
	outcome := metrics.IndexReportHandlingMessageToCentralSuccess
	defer func() {
		metrics.IndexReportProcessingDurationMilliseconds.
			WithLabelValues(outcome).
			Observe(metrics.StartTimeToMS(startTime))
	}()

	if indexReport == nil {
		outcome = metrics.IndexReportHandlingMessageToCentralNilReport
		log.Warn("Received nil virtual machine index report: not sending to Central")
		return
	}
	log.Debugf("Handling virtual machine index report with vsock_cid=%q...", indexReport.GetVsockCid())

	msg, outcome, err := h.newMessageToCentral(indexReport)
	if err != nil {
		// TODO: send a message the sensor relay to retry later if the VM was not found
		log.Warnf("unable to send index report message for the virtual machine with vsock cid %q to central: %v", indexReport.GetVsockCid(), err)
		return
	}
	h.sendIndexReportEvent(toCentral, msg)
	metrics.IndexReportsSent.With(metrics.StatusSuccessLabels).Inc()
}

func (h *handlerImpl) newMessageToCentral(indexReport *v1.IndexReport) (*message.ExpiringMessage, string, error) {
	cid, err := strconv.ParseUint(indexReport.GetVsockCid(), 10, 32)
	if err != nil {
		return nil, metrics.IndexReportHandlingMessageToCentralInvalidCID, errors.Wrapf(err, "Received an invalid Vsock CID: %q", indexReport.GetVsockCid())
	}

	vmInfo := h.store.GetFromCID(uint32(cid))
	if vmInfo == nil {
		// In test mode, auto-generate VMs on-the-fly when index reports arrive
		if env.IsVMTestModeEnabled() {
			vmInfo = h.generateAndStoreVM(uint32(cid))
			log.Infof("Auto-generated VM %s (name: %s, CID: %d) on-the-fly for load testing",
				vmInfo.ID, vmInfo.Name, *vmInfo.VSOCKCID)
		} else {
			// Return retryable error if the virtual machine is not yet known to Sensor.
			return nil, metrics.IndexReportHandlingMessageToCentralVMUnknown, errors.Wrapf(errVirtualMachineNotFound, "VirtualMachine with Vsock CID %q not found", indexReport.GetVsockCid())
		}
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

func (h *handlerImpl) sendIndexReportEvent(
	toCentral chan<- *message.ExpiringMessage,
	msg *message.ExpiringMessage,
) {
	select {
	case <-h.stopper.Flow().StopRequested():
	case toCentral <- msg:
	}
}

// generateAndStoreVM creates a new VM on-the-fly for load testing.
// This function generates a deterministic UUID based on the vsock CID and
// creates a minimal VM with the required fields, then adds it to the store.
func (h *handlerImpl) generateAndStoreVM(cid uint32) *virtualmachine.Info {
	const testNamespace = "vm-load-test"

	// Generate deterministic UUID based on CID for test mode
	// This ensures VMs with the same CID get the same UUID across restarts
	vmID := virtualmachine.VMID(generateDeterministicUUID(cid).String())
	vsockCID := new(uint32)
	*vsockCID = cid

	vmInfo := &virtualmachine.Info{
		ID:        vmID,
		Name:      fmt.Sprintf("vm-%d", cid),
		Namespace: testNamespace,
		VSOCKCID:  vsockCID,
		Running:   true,
		GuestOS:   "linux",
	}

	// Add to store and return the stored version
	return h.store.AddOrUpdate(vmInfo)
}

// generateDeterministicUUID creates a deterministic UUID v5 based on the vsock CID.
// This ensures the same CID always produces the same UUID, which is useful for
// test scenarios where VMs may be recreated with the same CID.
func generateDeterministicUUID(cid uint32) uuid.UUID {
	// Use a namespace UUID for StackRox VM test mode
	// This is a custom namespace UUID for "stackrox.io/vm-test"
	namespace := uuid.FromStringOrPanic("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	// Generate UUID v5 using the CID as the name
	return uuid.NewV5(namespace, fmt.Sprintf("vm-cid-%d", cid))
}
