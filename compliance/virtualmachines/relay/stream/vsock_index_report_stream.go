// Package stream manages the vsock server and produces validated index reports.
// It combines connection management, parsing, and validation before forwarding
// reports to the relay for sending to Sensor.
package stream

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/connutil"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/vsock"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
)

var log = logging.LoggerForModule()

type VsockIndexReportStream struct {
	listener                 net.Listener
	listenerMu               sync.Mutex
	semaphore                *semaphore.Weighted
	maxConcurrentConnections int
	semaphoreTimeout         time.Duration
	connectionReadTimeout    time.Duration
	waitAfterFailedAccept    time.Duration
	maxSizeBytes             int
}

// New creates a VsockIndexReportStream with a vsock listener.
// Concurrency limits are read from env vars VirtualMachinesMaxConcurrentVsockConnections
// and VirtualMachinesConcurrencyTimeout.
func New() (*VsockIndexReportStream, error) {
	listener, err := vsock.NewListener()
	if err != nil {
		return nil, errors.Wrap(err, "creating vsock listener")
	}

	maxConcurrentConnections := env.VirtualMachinesMaxConcurrentVsockConnections.IntegerSetting()
	semaphoreTimeout := env.VirtualMachinesConcurrencyTimeout.DurationSetting()
	maxSizeBytes := env.VirtualMachinesVsockConnMaxSizeKB.IntegerSetting() * 1024

	return &VsockIndexReportStream{
		listener:                 listener,
		semaphore:                semaphore.NewWeighted(int64(maxConcurrentConnections)),
		maxConcurrentConnections: maxConcurrentConnections,
		semaphoreTimeout:         semaphoreTimeout,
		connectionReadTimeout:    10 * time.Second,
		waitAfterFailedAccept:    time.Second,
		maxSizeBytes:             maxSizeBytes,
	}, nil
}

// Start begins accepting vsock connections and returns a channel of validated reports.
// The stream spawns goroutines to handle each connection concurrently (up to the
// configured limit). Reports are validated before being sent to the channel.
func (p *VsockIndexReportStream) Start(ctx context.Context) (<-chan *v1.VMReport, error) {
	log.Info("Starting report stream")

	if p.listener == nil {
		return nil, errors.New("listener is nil")
	}

	// Buffer size = concurrency limit to allow stream goroutines to complete
	// without blocking on sender. Use the already-derived maxConcurrentConnections
	// to keep this as the single source of truth.
	reportChan := make(chan *v1.VMReport, p.maxConcurrentConnections)

	// Single place that shuts down the listener when the context is done.
	go func() {
		<-ctx.Done()
		p.stop()
	}()

	// Start the accept loop in a goroutine
	go p.acceptLoop(ctx, reportChan)

	return reportChan, nil
}

func (p *VsockIndexReportStream) acceptLoop(ctx context.Context, reportChan chan<- *v1.VMReport) {
	for {
		// Accept() is blocking, but it will return when ctx is cancelled and the goroutine in Start() calls p.stop()
		conn, err := p.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				log.Info("Stopping report stream")
				return
			}

			// We deliberately don't kill the listener on errors. The only way to stop that is to cancel the context.
			// If we had return here on fatal errors, then compliance would continue working without the relay
			// and that would make it an invisible problem to the user.
			log.Errorf("Error accepting connection: %v", err)

			select {
			case <-time.After(p.waitAfterFailedAccept):
			case <-ctx.Done():
				return
			}
			continue
		}
		metrics.ConnectionsAccepted.Inc()

		if err := p.acquireSemaphore(ctx); err != nil {
			if ctx.Err() != nil {
				log.Info("Stopping report stream")
				return
			}

			// This log is rate-limited because when the concurrency limit is reached it is emitted every
			// semaphoreTimeout, which is user-configurable (min: 1 second).
			logging.GetRateLimitedLogger().WarnL(
				"relay semaphore timeout",
				"Failed to acquire semaphore to handle connection: %v",
				err,
			)

			// When the concurrency limit is reached, the semaphore cannot be acquired. We close the connection and
			// continue to listen. In this case, there is no need to add an extra wait to prevent a busy loop, because
			// we already waited semaphoreTimeout
			if err := conn.Close(); err != nil {
				log.Warnf("Failed to close connection after failing to acquire semaphore: %v", err)
			}

			continue
		}

		go p.handleConnection(ctx, conn, reportChan)
	}
}

func (p *VsockIndexReportStream) handleConnection(ctx context.Context, conn net.Conn, reportChan chan<- *v1.VMReport) {
	defer p.releaseSemaphore()

	defer func(conn net.Conn) {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}(conn)

	log.Infof("Handling connection from %s", conn.RemoteAddr())

	vmReport, err := p.receiveAndValidateVMReport(conn)
	if err != nil {
		log.Errorf("Error handling connection from %v: %v", conn.RemoteAddr(), err)
		return
	}

	log.Infof("Finished handling connection from %s", conn.RemoteAddr())

	// Send validated message to channel. Use select to avoid blocking during shutdown
	// when the relay stops reading from the channel.
	select {
	case reportChan <- vmReport:
		// Report sent successfully
	case <-ctx.Done():
		// Context cancelled during send - exit without blocking to allow defers to execute
		log.Debug("Context cancelled while sending report, skipping send")
		return
	}
}

func (p *VsockIndexReportStream) receiveAndValidateVMReport(conn net.Conn) (*v1.VMReport, error) {
	vsockCID, err := vsock.ExtractVsockCIDFromConnection(conn)
	if err != nil {
		return nil, errors.Wrap(err, "extracting vsock CID")
	}

	data, err := connutil.ReadFromConn(conn, p.maxSizeBytes, p.connectionReadTimeout)
	if err != nil {
		return nil, errors.Wrapf(err, "reading from connection (vsock CID: %d)", vsockCID)
	}

	log.Debugf("Parsing VM report (vsock CID: %d)", vsockCID)
	vmReport, err := parseVMReport(data)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing VM report data (vsock CID: %d)", vsockCID)
	}

	err = validateReportedVsockCID(vmReport, vsockCID)
	if err != nil {
		log.Debugf("Error validating reported vsock CID: %v", err)
		return nil, errors.Wrap(err, "validating reported vsock CID")
	}

	metrics.IndexReportsReceived.Inc()

	return vmReport, nil
}

func parseVMReport(data []byte) (*v1.VMReport, error) {
	msg := &v1.VMReport{}

	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, errors.Wrap(err, "unmarshalling data")
	}
	return msg, nil
}

// validateReportedVsockCID ensures the message's vsock CID matches the connection.
func validateReportedVsockCID(vmReport *v1.VMReport, connVsockCID uint32) error {
	reportedCID := vmReport.GetIndexReport().GetVsockCid()
	if reportedCID != strconv.FormatUint(uint64(connVsockCID), 10) {
		return errors.Errorf("mismatch between reported (%s) and real (%d) vsock CIDs", reportedCID, connVsockCID)
	}
	return nil
}

func (p *VsockIndexReportStream) stop() {
	p.listenerMu.Lock()
	defer p.listenerMu.Unlock()

	if p.listener == nil {
		return
	}

	log.Info("Stopping index report stream")
	if err := p.listener.Close(); err != nil {
		log.Errorf("Error closing listener: %v", err)
	}
	p.listener = nil
}

func (p *VsockIndexReportStream) acquireSemaphore(parentCtx context.Context) error {
	semCtx, cancel := context.WithTimeout(parentCtx, p.semaphoreTimeout)
	defer cancel()

	metrics.SemaphoreQueueSize.Inc()
	defer metrics.SemaphoreQueueSize.Dec()
	if err := p.semaphore.Acquire(semCtx, 1); err != nil {
		reason := "unknown"
		if errors.Is(err, context.DeadlineExceeded) {
			log.Debug("Could not acquire semaphore, too many concurrent connections")
			reason = "concurrency_limit"
		} else if errors.Is(err, context.Canceled) {
			log.Debug("Could not acquire semaphore, the context was canceled")
			reason = "context_canceled"
		}
		metrics.SemaphoreAcquisitionFailures.With(prometheus.Labels{"reason": reason}).Inc()
		return errors.Wrap(err, "failed to acquire semaphore")
	}
	metrics.SemaphoreHoldingSize.Inc()
	return nil
}

func (p *VsockIndexReportStream) releaseSemaphore() {
	p.semaphore.Release(1)
	metrics.SemaphoreHoldingSize.Dec()
}
