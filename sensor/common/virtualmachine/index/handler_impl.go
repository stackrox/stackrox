package index

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
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
	toCompliance chan common.MessageToComplianceWithAddress
	indexReports chan *v1.IndexReport
	store        VirtualMachineStore

	// reactiveMu guards reactivePending. Separate from lock (which guards
	// Start/Stop/Send channel lifecycle) since it protects independent state
	// that is accessed far more frequently.
	reactiveMu sync.Mutex
	// reactivePending holds at most one not-yet-delivered reactive report
	// per VM ID. A reactive send for a VM that already has a pending entry
	// replaces it — only the newest report is ever meaningful. Bounded by
	// the number of VMs Sensor knows about, not an arbitrary constant.
	reactivePending map[virtualmachine.VMID]*reactiveEntry
	// reactiveWake signals the run() loop that reactivePending has a new
	// entry. Buffered size 1 with non-blocking sends: a pending wake-up
	// already covers any further upserts until it's consumed.
	reactiveWake chan struct{}
}

// reactiveEntry pairs a reactive report with the roxagent-reported scan time
// used for SLA latency measurement (never forwarded to Central).
type reactiveEntry struct {
	report      *v1.IndexReport
	generatedAt time.Time
}

func (h *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.SensorACKSupport}
}

func (h *handlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return h.stopper.Client().Stopped()
}

// ComplianceC returns a channel with messages destined for Compliance.
func (h *handlerImpl) ComplianceC() <-chan common.MessageToComplianceWithAddress {
	return h.toCompliance
}

func (h *handlerImpl) Send(ctx context.Context, vm *v1.IndexReport) error {
	if err := h.checkSendPreconditions(vm.GetVsockCid()); err != nil {
		return err
	}

	h.lock.RLock()
	defer h.lock.RUnlock()

	return h.enqueueScheduled(ctx, vm)
}

// SendReactive enqueues a report produced by a reactive (event-triggered)
// rescan with priority over Send. At most one pending reactive report is
// kept per VM: a newer report for the same VM replaces an older,
// not-yet-delivered one rather than queueing alongside it, since only the
// latest report is ever meaningful (mirrors the generation-counter semantics
// already in the VSOCK protocol).
func (h *handlerImpl) SendReactive(ctx context.Context, vm *v1.IndexReport, generatedAt time.Time) error {
	if err := h.checkSendPreconditions(vm.GetVsockCid()); err != nil {
		return err
	}

	h.lock.RLock()
	defer h.lock.RUnlock()

	vmID, err := h.resolveVMIDForReactive(vm.GetVsockCid())
	if err != nil {
		// VM identity isn't resolvable yet (e.g. just connected). Fall back
		// to the normal path instead of silently dropping a reactive update
		// — it will still be retried on the next scheduled poll either way.
		log.Warnf("Reactive index report for vsock_cid=%q could not be prioritized (%v); using the normal queue instead", vm.GetVsockCid(), err)
		return h.enqueueScheduled(ctx, vm)
	}

	concurrency.WithLock(&h.reactiveMu, func() {
		if h.reactivePending == nil {
			h.reactivePending = make(map[virtualmachine.VMID]*reactiveEntry)
		}
		h.reactivePending[vmID] = &reactiveEntry{report: vm, generatedAt: generatedAt}
	})

	select {
	case h.reactiveWake <- struct{}{}:
	default:
	}
	return nil
}

func (h *handlerImpl) checkSendPreconditions(vsockCID string) error {
	if h.stopper.Client().Stopped().IsDone() {
		return errox.InvariantViolation.CausedBy(errInputChanClosed)
	}
	if !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
		return errox.NotImplemented.CausedBy(errCapabilityNotSupported)
	}
	if !h.centralReady.IsDone() {
		log.Warnf("Cannot send index report for virtual machine with vsock_cid=%q to Central because Central is not reachable", vsockCID)
		metrics.IndexReportsSent.With(metrics.StatusCentralNotReadyLabels).Inc()
		return errox.ResourceExhausted.CausedBy(errCentralNotReachable)
	}
	return nil
}

// enqueueScheduled is the original Send() body: enqueues onto the bounded
// indexReports channel, blocking up to ctx's deadline and recording
// backpressure metrics.
func (h *handlerImpl) enqueueScheduled(ctx context.Context, vm *v1.IndexReport) error {
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

// resolveVMIDForReactive looks up the VM identity for a reactive report so it
// can be prioritized per-VM. Deliberately kept separate from
// newMessageToCentral's resolution (used at dequeue time for every report,
// with its own outcome-metric labels) to avoid touching that already-tested
// path.
func (h *handlerImpl) resolveVMIDForReactive(vsockCIDStr string) (virtualmachine.VMID, error) {
	cid, err := strconv.ParseUint(vsockCIDStr, 10, 32)
	if err != nil {
		return "", errors.Wrapf(err, "invalid vsock CID %q", vsockCIDStr)
	}
	vmInfo := h.store.GetFromCID(uint32(cid))
	if vmInfo == nil {
		return "", errors.Wrapf(errVirtualMachineNotFound, "vsock CID %q", vsockCIDStr)
	}
	return vmInfo.ID, nil
}

// popReactivePending removes and returns one arbitrary entry from
// reactivePending, if any. Go map iteration order is randomized, which is
// fine here: with at most one entry per VM and no ordering promised between
// different VMs' reactive updates, any order is correct.
func (h *handlerImpl) popReactivePending() (*reactiveEntry, bool) {
	return concurrency.WithLock2(&h.reactiveMu, func() (*reactiveEntry, bool) {
		for vmID, entry := range h.reactivePending {
			delete(h.reactivePending, vmID)
			return entry, true
		}
		return nil, false
	})
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
func (h *handlerImpl) ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error {
	sensorAck := msg.GetSensorAck()
	if sensorAck == nil || sensorAck.GetMessageType() != central.SensorACK_VM_INDEX_REPORT {
		return nil
	}

	vmID := sensorAck.GetResourceId()
	action := sensorAck.GetAction()
	reason := sensorAck.GetReason()
	h.forwardToCompliance(ctx, vmID, action, reason)

	// Not limiting to ACK & NACK and recording all types of actions for better debuggability.
	// The risk of prometheus label cardinality explosion is considered low and accepted hereby.
	metrics.IndexReportAcksReceived.WithLabelValues(action.String()).Inc()

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
	if h.toCentral != nil || h.indexReports != nil || h.toCompliance != nil {
		return errStartMoreThanOnce
	}
	h.indexReports = make(chan *v1.IndexReport, env.VirtualMachinesIndexReportsBufferSize.IntegerSetting())
	h.toCompliance = make(chan common.MessageToComplianceWithAddress, 1)
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
			// Priority path: always drain a pending reactive report before
			// considering anything from the routine indexReports channel.
			if entry, ok := h.popReactivePending(); ok {
				h.handleIndexReport(ch2Central, entry.report, entry.generatedAt)
				continue
			}
			select {
			case <-h.stopper.Flow().StopRequested():
				return
			case <-h.reactiveWake:
				// A reactive report just arrived; loop back to the top so
				// the priority check above drains it before indexReports.
			case indexReport, ok := <-indexReports:
				if !ok {
					h.stopper.Flow().StopWithError(errInputChanClosed)
					return
				}
				h.handleIndexReport(ch2Central, indexReport, time.Time{})
			}
		}
	}()
	return ch2Central
}

func (h *handlerImpl) forwardToCompliance(
	ctx context.Context,
	resourceID string,
	action central.SensorACK_Action,
	reason string,
) {
	if h.toCompliance == nil {
		log.Debug("Compliance channel not initialized; skipping forwarding VM ACK/NACK")
		return
	}

	var complianceAction sensor.MsgToCompliance_ComplianceACK_Action
	switch action {
	case central.SensorACK_ACK:
		complianceAction = sensor.MsgToCompliance_ComplianceACK_ACK
	case central.SensorACK_NACK:
		complianceAction = sensor.MsgToCompliance_ComplianceACK_NACK
	default:
		log.Warnf("Unknown SensorACK action for VM index report: %v", action)
		return
	}

	// Resolve the VM's host node so the compliance multiplexer can route to
	// the correct compliance connection.
	var nodeName string
	if resourceID != "" {
		vmID := vmIDFromResourceID(resourceID)
		if vmInfo := h.store.Get(virtualmachine.VMID(vmID)); vmInfo != nil {
			nodeName = vmInfo.NodeName
		}
	}

	// Drop the ACK when the target node is unknown rather than broadcasting.
	// vsock CIDs are host-local and can collide across nodes; broadcasting
	// would risk a compliance instance on another node matching the CID to
	// the wrong VM. A deleted/migrated VM does not need its ACK delivered.
	if nodeName == "" {
		log.Debugf("Dropping VM ACK/NACK (resourceID=%s): VM not found in store, cannot route to node", resourceID)
		return
	}

	msg := common.MessageToComplianceWithAddress{
		Msg: &sensor.MsgToCompliance{
			Msg: &sensor.MsgToCompliance_ComplianceAck{
				ComplianceAck: &sensor.MsgToCompliance_ComplianceACK{
					Action:      complianceAction,
					MessageType: sensor.MsgToCompliance_ComplianceACK_VM_INDEX_REPORT,
					ResourceId:  resourceID,
					Reason:      reason,
				},
			},
		},
		// Hostname is the node name (not the VM resource ID) used by the
		// compliance multiplexer to route to the correct compliance connection.
		Hostname:  nodeName,
		Broadcast: false,
	}

	select {
	case <-ctx.Done():
		log.Warnf("Dropping VM ACK/NACK (resourceID=%s): %v", resourceID, ctx.Err())
		return
	case <-h.stopper.Flow().StopRequested():
		log.Debugf("Dropping VM ACK/NACK (resourceID=%s) during shutdown", resourceID)
		return
	default:
	}

	select {
	case h.toCompliance <- msg:
	default:
		log.Warnf("Dropping VM ACK/NACK (resourceID=%s): compliance queue is full", resourceID)
	}
}

func (h *handlerImpl) handleIndexReport(
	toCentral chan *message.ExpiringMessage,
	indexReport *v1.IndexReport,
	generatedAt time.Time,
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
	log.Debugf("Handling virtual machine index report (vm_id=%q vsock_cid=%q)...",
		indexReport.GetVmId(), indexReport.GetVsockCid())

	msg, outcome, err := h.newMessageToCentral(indexReport)
	if err != nil {
		// TODO: send a message the sensor relay to retry later if the VM was not found
		log.Warnf("unable to send index report message for virtual machine (vm_id=%q vsock_cid=%q) to central: %v",
			indexReport.GetVmId(), indexReport.GetVsockCid(), err)
		return
	}
	h.sendIndexReportEvent(toCentral, msg)
	metrics.IndexReportsSent.With(metrics.StatusSuccessLabels).Inc()

	if !generatedAt.IsZero() {
		metrics.VirtualMachineReactiveIndexReportLatencySeconds.Observe(time.Since(generatedAt).Seconds())
	}
}

func (h *handlerImpl) newMessageToCentral(indexReport *v1.IndexReport) (*message.ExpiringMessage, string, error) {
	vmInfo, outcome, err := h.resolveVM(indexReport)
	if err != nil {
		return nil, outcome, err
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

// resolveVM finds the VirtualMachine for an index report. Prefer vm_id when
// Sensor already knows the Kubernetes UID (pull path). Otherwise resolve via
// vsock_cid (push path from agent/relay).
func (h *handlerImpl) resolveVM(indexReport *v1.IndexReport) (*virtualmachine.Info, string, error) {
	if vmID := indexReport.GetVmId(); vmID != "" {
		vmInfo := h.store.Get(virtualmachine.VMID(vmID))
		if vmInfo == nil {
			return nil, metrics.IndexReportHandlingMessageToCentralVMUnknown,
				errors.Wrapf(errVirtualMachineNotFound, "VirtualMachine %q not found", vmID)
		}
		return vmInfo, "", nil
	}

	cid, err := strconv.ParseUint(indexReport.GetVsockCid(), 10, 32)
	if err != nil {
		return nil, metrics.IndexReportHandlingMessageToCentralInvalidCID,
			errors.Wrapf(err, "Received an invalid Vsock CID: %q", indexReport.GetVsockCid())
	}

	vmInfo := h.store.GetFromCID(uint32(cid))
	if vmInfo == nil {
		return nil, metrics.IndexReportHandlingMessageToCentralVMUnknown,
			errors.Wrapf(errVirtualMachineNotFound, "VirtualMachine with Vsock CID %q not found", indexReport.GetVsockCid())
	}
	return vmInfo, "", nil
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

// vmIDFromResourceID extracts the VM ID from a composite ACK resource ID
// (format "vmID:vsockCID"). Returns the input as-is when no separator is found.
func vmIDFromResourceID(resourceID string) string {
	if before, _, ok := strings.Cut(resourceID, ":"); ok {
		return before
	}
	return resourceID
}
