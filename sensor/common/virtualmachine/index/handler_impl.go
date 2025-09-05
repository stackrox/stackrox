package index

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
)

// TODO: Buffer has been decreased for testing. Increase the buffer again.
const indexReportsBufferedChannelSize = 1

var (
	errCapabilityNotSupported = errors.New("Central does not have virtual machine capability")
	errCentralNotReachable    = errors.New("Central is not reachable")
	errInputChanClosed        = errors.New("channel receiving virtual machines is closed")
	errStartMoreThanOnce      = errors.New("unable to start the handler more than once")
)

type handlerImpl struct {
	centralReady concurrency.Signal
	// lock prevents the race condition between Start() [writer] and ResponsesC(), Send() [reader].
	lock         *sync.RWMutex
	stopper      concurrency.Stopper
	toCentral    <-chan *message.ExpiringMessage
	indexReports chan *v1.IndexReport
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
	select {
	case <-ctx.Done():
		if err := ctx.Err(); errors.Is(err, context.DeadlineExceeded) {
			metrics.IndexReportsSent.With(metrics.StatusTimeoutLabels).Inc()
			return err //nolint:wrapcheck
		}
		metrics.IndexReportsSent.With(metrics.StatusErrorLabels).Inc()
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

func (h *handlerImpl) Filter(msg *central.MsgToSensor) bool {
	return false
}

// ProcessMessage is a no-op because Sensor does not receive any virtual machine data
// from Central.
func (h *handlerImpl) ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error {
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
	h.indexReports = make(chan *v1.IndexReport, indexReportsBufferedChannelSize)
	h.toCentral = h.run()
	return nil
}

func (h *handlerImpl) Stop() {
	close(h.indexReports)
	if !h.stopper.Client().Stopped().IsDone() {
		defer utils.IgnoreError(h.stopper.Client().Stopped().Wait)
	}
	h.stopper.Client().Stop()
}

// run handles the virtual machine data and forwards it to Central.
// This is the only goroutine that writes into the toCentral channel, thus it is
// responsible for creating and closing that chan.
func (h *handlerImpl) run() (toCentral <-chan *message.ExpiringMessage) {
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
			case indexReport, ok := <-h.indexReports:
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
	log.Debugf("Handling virtual machine index report with vsock_cid=%q...", indexReport.GetVsockCid())
	if indexReport == nil {
		log.Warn("Received nil virtual machine index report: not sending to Central")
		return
	}

	h.sendIndexReportEvent(toCentral, indexReport)
	metrics.IndexReportsSent.With(metrics.StatusSuccessLabels).Inc()
}

func (h *handlerImpl) sendIndexReportEvent(
	toCentral chan<- *message.ExpiringMessage,
	indexReport *v1.IndexReport,
) {
	if indexReport == nil {
		return
	}
	select {
	case <-h.stopper.Flow().StopRequested():
	case toCentral <- message.New(&central.MsgFromSensor{
		// TODO: Look up the actual Virtual Machine ID from the informer store.
		// Return retryable error if the virtual machine is not yet known to Sensor.
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     indexReport.GetVsockCid(),
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachineIndexReport{
					VirtualMachineIndexReport: &v1.IndexReportEvent{
						// TODO: VsockCid is used as a dummy here.
						Id:    indexReport.GetVsockCid(),
						Index: indexReport,
					},
				},
			},
		},
	}):
	}
}
