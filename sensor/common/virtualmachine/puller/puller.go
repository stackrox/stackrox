package puller

import (
	"context"
	"strconv"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
)

var log = logging.LoggerForModule()

const defaultPollInterval = 1 * time.Minute

// RunningVMStore provides the list of running VMs.
type RunningVMStore interface {
	ListRunning() []*virtualmachine.Info
}

// VMDialer connects to a VM's VSOCK port across namespaces.
type VMDialer interface {
	Dial(namespace, name string, port uint32, useTLS bool) (vsockclient.StreamReader, error)
}

// IndexReportSender sends index reports toward Central.
type IndexReportSender interface {
	Send(ctx context.Context, report *v1.IndexReport) error
}

var _ common.SensorComponent = (*Puller)(nil)

// Puller periodically pulls VM scan reports from running VMs via VSOCK.
// It implements common.SensorComponent for proper lifecycle management.
type Puller struct {
	store    RunningVMStore
	sender   IndexReportSender
	dialer   VMDialer
	interval time.Duration
	port     uint32
	stopper  concurrency.Stopper
}

// New creates a Puller. The dialer must support cross-namespace dialing.
func New(store RunningVMStore, sender IndexReportSender, dialer VMDialer) *Puller {
	return &Puller{
		store:    store,
		sender:   sender,
		dialer:   dialer,
		interval: defaultPollInterval,
		port:     vsockclient.DefaultVSOCKPort,
		stopper:  concurrency.NewStopper(),
	}
}

func (p *Puller) Name() string { return "virtualmachine.puller" }

func (p *Puller) Start() error {
	log.Infof("VSOCK puller component starting (interval=%s, port=%d)", p.interval, p.port)
	go p.run()
	return nil
}

func (p *Puller) Stop() {
	log.Info("VSOCK puller component stopping")
	p.stopper.Client().Stop()
}

func (p *Puller) Capabilities() []centralsensor.SensorCapability { return nil }

func (p *Puller) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, p.Name()))
}

func (p *Puller) ResponsesC() <-chan *message.ExpiringMessage { return nil }

func (p *Puller) ProcessMessage(_ context.Context, _ *central.MsgToSensor) error { return nil }

func (p *Puller) Accepts(_ *central.MsgToSensor) bool { return false }

func (p *Puller) run() {
	defer p.stopper.Flow().ReportStopped()

	log.Infof("VSOCK pull-mode poller goroutine started (interval=%s, port=%d)", p.interval, p.port)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	log.Info("VSOCK pull: performing initial poll")
	p.poll()

	for {
		select {
		case <-p.stopper.Flow().StopRequested():
			log.Info("VSOCK pull-mode poller received stop signal, exiting")
			return
		case <-ticker.C:
			log.Info("VSOCK pull: tick — starting poll cycle")
			p.poll()
		}
	}
}

func (p *Puller) poll() {
	pollStart := time.Now()
	metrics.PullCyclesTotal.Inc()

	running := p.store.ListRunning()
	if len(running) == 0 {
		log.Info("VSOCK pull: no running VMs found in store, skipping poll cycle")
		metrics.PullCycleDurationMilliseconds.Observe(metrics.StartTimeToMS(pollStart))
		return
	}

	log.Infof("VSOCK pull: found %d running VM(s), starting pull cycle", len(running))

	var succeeded, failed int
	for _, vm := range running {
		if p.pullOne(vm) {
			succeeded++
		} else {
			failed++
		}
	}

	cycleDuration := metrics.StartTimeToMS(pollStart)
	metrics.PullCycleDurationMilliseconds.Observe(cycleDuration)

	log.Infof("VSOCK pull: poll cycle complete — %d succeeded, %d failed out of %d VMs (%.1fms)",
		succeeded, failed, len(running), cycleDuration)
}

func (p *Puller) pullOne(vm *virtualmachine.Info) bool {
	vmLabel := vm.Namespace + "/" + vm.Name
	log.Infof("VSOCK pull [%s]: dialing VSOCK port %d (vm_id=%s, vsock_cid=%v)",
		vmLabel, p.port, vm.ID, formatCID(vm.VSOCKCID))

	dialStart := time.Now()
	stream, err := p.dialer.Dial(vm.Namespace, vm.Name, p.port, false)
	dialDuration := metrics.StartTimeToMS(dialStart)

	if err != nil {
		metrics.PullDialErrorsTotal.Inc()
		log.Warnf("VSOCK pull [%s]: dial failed after %.1fms: %v", vmLabel, dialDuration, err)
		return false
	}
	log.Infof("VSOCK pull [%s]: dial succeeded in %.1fms, reading report...", vmLabel, dialDuration)

	readStart := time.Now()
	report, err := vsockclient.ReadVMReport(stream)
	readDuration := metrics.StartTimeToMS(readStart)

	if err != nil {
		metrics.PullReadErrorsTotal.Inc()
		log.Warnf("VSOCK pull [%s]: read failed after %.1fms: %v", vmLabel, readDuration, err)
		return false
	}

	pkgCount := len(report.GetIndexReport().GetIndexV4().GetContents().GetPackages())
	reportCID := report.GetIndexReport().GetVsockCid()
	log.Infof("VSOCK pull [%s]: report received in %.1fms (vsock_cid=%s, packages=%d, success=%v)",
		vmLabel, readDuration, reportCID, pkgCount, report.GetIndexReport().GetIndexV4().GetSuccess())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sendStart := time.Now()
	if err := p.sender.Send(ctx, report.GetIndexReport()); err != nil {
		metrics.PullSendErrorsTotal.Inc()
		log.Warnf("VSOCK pull [%s]: failed to enqueue report for Central after %.1fms: %v",
			vmLabel, metrics.StartTimeToMS(sendStart), err)
		return false
	}

	totalDuration := metrics.StartTimeToMS(dialStart)
	metrics.PullReportsReceivedTotal.Inc()
	metrics.PullDurationMilliseconds.Observe(totalDuration)

	log.Infof("VSOCK pull [%s]: report enqueued for Central (total=%.1fms, dial=%.1fms, read=%.1fms, send=%.1fms)",
		vmLabel, totalDuration, dialDuration, readDuration, metrics.StartTimeToMS(sendStart))
	return true
}

func formatCID(cid *uint32) string {
	if cid == nil {
		return "<nil>"
	}
	return strconv.FormatUint(uint64(*cid), 10)
}
